package main

import (
	"github.com/go-kit/kit/log/level"
	"fmt"
	"time"
	"database/sql"
	"sort"
	"bitbucket.org/garyyu/algo-trading/go-binance"
)

var (
	LatestOrderID = make(map[string]int64)
)

type OrderData struct {
	id				 		 int64		`json:"id"`
	ProjectID				 int64		`json:"ProjectID"`
	IsDone					 bool		`json:"IsDone"`
	executedOrder			 binance.ExecutedOrder	`json:"executedOrder"`
	LastQueryTime            time.Time	`json:"LastQueryTime"`
}

/*
 * Expensive API: Weight: 5
 * It help to import all orders into local database, and map them into related projects.
 * 		Only call it from Project Manager
 */
func GetAllOrders(symbol string){
	//
	//fmt.Println("GetAllOrders func enter for", symbol)

	// find active project in same asset
	var project *ProjectData=nil

	//ProjectMutex.RLock()	//--- must not lock here! because the caller ProjectManager() already Lock().
	for _, activeProject := range ActiveProjectList {
		if activeProject.Symbol == symbol {
			project = activeProject
			break
		}
	}
	//ProjectMutex.RUnlock()

	if project==nil{
		fmt.Println("GetAllOrders - Fail to find ProjectID for Symbol", symbol)
		return
	}

	oldLatestOrderID,ok := LatestTradeID[symbol]
	if !ok{
		LatestTradeID[symbol] = 0
		oldLatestOrderID = 0
	}
	keepOldOne := false

	executedOrderList, err := binanceSrv.AllOrders(binance.AllOrdersRequest{
		Symbol:     symbol,
		OrderID:	LatestTradeID[symbol],
		RecvWindow: 5 * time.Second,
		Timestamp:  time.Now(),
	})
	if err != nil {
		level.Error(logger).Log("GetAllOrders - fail! Symbol=", symbol)
		return
	}
	//
	//fmt.Printf("GetAllOrders - Return %d Orders\n", len(executedOrderList))

	// Sort by Time, in case Binance does't sort them
	sort.Slice(executedOrderList, func(m, n int) bool {
		return executedOrderList[m].Time.Before(executedOrderList[n].Time)
	})

	var newOrdersImported = 0
	for _, executedOrder := range executedOrderList {
		//
		//fmt.Printf("GetAllOrders - Get Order: %v\n", executedOrder)

		if executedOrder.OrderID > LatestTradeID[symbol] {
			LatestTradeID[symbol] = executedOrder.OrderID
		}

		// check if this order is done
		IsDone := false
		if executedOrder.Status == binance.StatusPartiallyFilled ||
			executedOrder.Status == binance.StatusNew {
			IsDone = false
		} else if executedOrder.IsWorking {
			IsDone = true
		}

		// already in local database?
		if GetOrderId(executedOrder.OrderID)>0 {
			continue
		}

		// insert data into order list
		orderData := OrderData{
			ProjectID: -1,
			executedOrder:*executedOrder,
			IsDone: IsDone,
		}
		if InsertOrder(&orderData)<=0 {
			fmt.Println("GetAllOrders - InsertOrder Fail. Order=", orderData)
			keepOldOne = true
		}else{
			newOrdersImported += 1
		}
	}

	// in case we fail to save to local database
	if keepOldOne {
		LatestTradeID[symbol] = oldLatestOrderID
	}

	// try to map new imported orders to the project
	MatchProjectForOrder(project)
	//
	//fmt.Println("GetAllOrders func exit")
}

/*
 * For new imported orders, we have to solve which project this order belongs to, then
 * assign ProjectID to the order.
 */
func MatchProjectForOrder(project *ProjectData){
	//
	//fmt.Println("MatchProjectForOrder - func enter. Project=", project.Symbol)

	// Get recent trades list in this asset, with the order of latest first.
	orderList,isOldProject := getRecentOrderList(project.Symbol, binance.Day, project.id)

	// Not a new project? Let's put the ProjectID into all those new orders in same asset
	if isOldProject {

		for _,order := range orderList {

			if order.ProjectID >= 0 {
				break	// break here to avoid pollute very old orders record.
			}
			order.ProjectID = project.id
			if !UpdateOrderProjectID(&order){
				fmt.Printf("UpdateOrderProjectID - Fail. ProjectID=%d, order%v\n",
					project.id, order)
			}
		}

		return
	}

	amount := 0.0
	invest := 0.0

	ordersNum := 0
	for _,order := range orderList {
		ordersNum += 1

		if order.executedOrder.Side == binance.SideBuy {
			amount += order.executedOrder.ExecutedQty
			invest += order.executedOrder.ExecutedQty * order.executedOrder.Price
		}else{
			amount -= order.executedOrder.ExecutedQty
			invest -= order.executedOrder.ExecutedQty * order.executedOrder.Price
		}
		//
		//fmt.Printf("MatchProjectForOrder - %d: amount=%f, project InitialAmount=%f\n",
		//	i, amount, project.InitialAmount)

		if FloatEquals(amount, project.InitialAmount) {
			break
		}
	}

	fmt.Printf("MatchProjectForOrder - %s: amount=%f, project InitialAmount=%f\n",
		project.Symbol, amount, project.InitialAmount)

	// We find it? Let's put the ProjectID into all these orders
	if FloatEquals(amount, project.InitialAmount) {

		for i:=0; i<ordersNum; i++{
			order := orderList[i]
			order.ProjectID = project.id

			if !UpdateOrderProjectID(&order){
				fmt.Println("MatchProjectForOrder - UpdateOrderProjectID Failed. order:", order)
			}
		}

		// in case trades not downloaded yet
		if project.InitialBalance == 0{

			project.InitialBalance = invest
			project.InitialPrice = invest / amount		// average price
			if !UpdateProjectInitialBalance(project){
				fmt.Println("MatchProjectForOrder - Update Project InitialBalane Failed. InitialBalance",
					project.InitialBalance)
			}
		}
	}else{
		fmt.Println("MatchProjectForOrder - Warning! new project for asset", project.Symbol,
			"not found in my orders history! Project can't be managed.")
	}
}

func QueryOrders(){

	// Get Order List from local database.
	openOrderList := GetOrderList(false, -1)

	for _, openOrder := range openOrderList {

		if openOrder.IsDone {
			continue
		}

		executedOrder, err := binanceSrv.QueryOrder(binance.QueryOrderRequest{
			Symbol:     openOrder.executedOrder.Symbol,
			OrderID: openOrder.executedOrder.OrderID,
			RecvWindow: 5 * time.Second,
			Timestamp:  time.Now(),
		})
		if err != nil {
			level.Error(logger).Log("QueryOrder - fail! Symbol=", openOrder.executedOrder.Symbol,
				"OrderID=", openOrder.executedOrder.OrderID)
			return
		}

		openOrder.executedOrder = *executedOrder

		// check if this order is done
		if executedOrder.Status == binance.StatusPartiallyFilled ||
			executedOrder.Status == binance.StatusNew {
			openOrder.IsDone = false
		} else if executedOrder.IsWorking {
			openOrder.IsDone = true
		}

		if !UpdateOrder(&openOrder){
			fmt.Println("UpdateOrder - Failed. openOrder=", openOrder)
		}

		//fmt.Println("QueryOrders - Result:", executedOrder)
	}

}

func CancelOpenOrders(){

	ProjectMutex.RLock()
	defer ProjectMutex.RUnlock()

	for _, project := range ActiveProjectList {

		// only new order needs query
		if project.OrderStatus!=string(binance.StatusNew) {
			continue
		}

		cancelOrder(project.Symbol, project.OrderID)
	}
}

func cancelOrder(symbol string, OrderID int64){

	executedOrderList, err := binanceSrv.OpenOrders(binance.OpenOrdersRequest{
		Symbol:     symbol,
		RecvWindow: 5 * time.Second,
		Timestamp:  time.Now(),
	})
	if err != nil {
		level.Error(logger).Log("CancelOpenOrders - OpenOrders Query fail! Symbol=", symbol)
		return
	}

	for _, executedOrder := range executedOrderList {
		fmt.Println("OpenOrders - ", executedOrder)
	}

	// cancel remaining open orders

	canceledOrder, err := binanceSrv.CancelOrder(binance.CancelOrderRequest{
		Symbol:    symbol,
		OrderID:   OrderID,
		Timestamp: time.Now(),
	})
	if err != nil {
		level.Error(logger).Log("CancelOrder - fail! Symbol:", symbol, "error:", err)
		return
	}
	fmt.Println("CanceledOrder :", canceledOrder)
}


/*
 * Insert Order data into Database
 */
func InsertOrder(orderData *OrderData) int64{
	//
	//fmt.Printf("InsertOrder - %v", orderData)

	//
	//if orderData==nil || len(orderData.executedOrder.ClientOrderID)==0 {
	//	level.Warn(logger).Log("InsertOrder - invalid orderData!", orderData)
	//	return -1
	//}

	query := `INSERT INTO order_list (
				ProjectID, IsDone, Symbol, OrderID, ClientOrderID, Price, 
				OrigQty, ExecutedQty, Status, TimeInForce, Type, Side, 
				StopPrice, IcebergQty, Time, IsWorking, LastQueryTime 
			  ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NOW())`

	executedOrder := &orderData.executedOrder
	res, err := DBCon.Exec(query,
		orderData.ProjectID,
		orderData.IsDone,
		executedOrder.Symbol,
		executedOrder.OrderID,
		executedOrder.ClientOrderID,
		executedOrder.Price,
		executedOrder.OrigQty,
		executedOrder.ExecutedQty,
		string(executedOrder.Status),
		string(executedOrder.TimeInForce),
		string(executedOrder.Type),
		string(executedOrder.Side),
		executedOrder.StopPrice,
		executedOrder.IcebergQty,
		executedOrder.Time,
		executedOrder.IsWorking,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return -1
	}

	id, _ := res.LastInsertId()
	return id
}


/*
 * Insert Draft Order data into Database
 */
func InsertDraftOrder(orderData *OrderData) int64{
	//
	//fmt.Printf("InsertDraftOrder - %v", orderData)

	//
	//if orderData==nil || len(orderData.executedOrder.ClientOrderID)==0 {
	//	level.Warn(logger).Log("InsertOrder - invalid orderData!", orderData)
	//	return -1
	//}

	query := `INSERT INTO order_list (
				ProjectID, Symbol, OrderID, ClientOrderID, Price, OrigQty, Status, TimeInForce, Type, Side
			  ) VALUES (?,?,?,?,?,?,?,?,?,?)`

	executedOrder := &orderData.executedOrder
	res, err := DBCon.Exec(query,
		orderData.ProjectID,
		executedOrder.Symbol,
		executedOrder.OrderID,
		executedOrder.ClientOrderID,
		executedOrder.Price,
		executedOrder.OrigQty,
		"UNK","UNK","UNK","UNK",
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return -1
	}

	id, _ := res.LastInsertId()
	return id
}



/*
 * Used for detect if order exist in local database
 */
func GetOrderId(OrderID int64) int64 {

	row := DBCon.QueryRow("SELECT id FROM order_list WHERE OrderID=?", OrderID)

	var id int64 = -1
	err := row.Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		level.Error(logger).Log("GetOrderId - Scan Err:", err)
	}

	return id
}


/*
 * Update ProjectID for Order into Database
 */
func UpdateOrderProjectID(order *OrderData) bool{

	query := `UPDATE order_list SET ProjectID=? WHERE id=?`

	res, err := DBCon.Exec(query,
		order.ProjectID,
		order.id,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return false
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected>=0 {
		return true
	}else{
		return false
	}
}


/*
 * Update Order query result into Database
 */
func UpdateOrder(orderData *OrderData) bool{

	if orderData==nil || len(orderData.executedOrder.ClientOrderID)==0 || orderData.executedOrder.OrderID==0 {
		level.Warn(logger).Log("UpdateOrder - Invalid OrderData", orderData)
		return false
	}

	executedOrder := &orderData.executedOrder

	query := `UPDATE order_list SET IsDone=?, ClientOrderID=?, Price=?, OrigQty=?, 
				ExecutedQty=?, Status=?, TimeInForce=?, Type=?, Side=?, StopPrice=?,
				IcebergQty=?, Time=?, LastQueryTime=NOW() WHERE OrderID=?`

	res, err := DBCon.Exec(query,
		orderData.IsDone,
		executedOrder.ClientOrderID,
		executedOrder.Price,
		executedOrder.OrigQty,
		executedOrder.ExecutedQty,
		string(executedOrder.Status),
		string(executedOrder.TimeInForce),
		string(executedOrder.Type),
		string(executedOrder.Side),
		executedOrder.StopPrice,
		executedOrder.IcebergQty,
		executedOrder.Time,
		executedOrder.OrderID,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return false
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected>=0 {
		return true
	}else{
		return false
	}
}

/*
 * Get Order List from local database.
 * 	- IsDone=1 means orders already done.
 * 	- IsDone=0 means 'Open', in local database view, if status pending to update from Binance server.
 *  To get status, query the Binance server via API.
 */
func GetOrderList(isDone bool, projectId int64) []OrderData {

	var query string
	if isDone{
		query = "SELECT * FROM order_list WHERE IsDone=1 and ProjectID=" + fmt.Sprint(projectId)
	}else{
		query = "SELECT * FROM order_list WHERE IsDone=0 LIMIT 50"
	}
	rows, err := DBCon.Query(query)

	if err != nil {
		level.Error(logger).Log("getOrderList - DB.Query Fail. Err=", err)
		panic(err.Error())
	}
	defer rows.Close()

	OpenOrderList := make([]OrderData, 0)

rowLoopOpenOrder:
	for rows.Next() {

		var transactTime NullTime
		var LastQueryTime NullTime

		orderData := OrderData{}
		executedOrder := &orderData.executedOrder

		err := rows.Scan(&orderData.id, &orderData.ProjectID, &orderData.IsDone,
			&executedOrder.Symbol, &executedOrder.OrderID, &executedOrder.ClientOrderID,
			&executedOrder.Price, &executedOrder.OrigQty, &executedOrder.ExecutedQty,
			&executedOrder.Status, &executedOrder.TimeInForce, &executedOrder.Type,
			&executedOrder.Side, &executedOrder.StopPrice, &executedOrder.IcebergQty,
			&transactTime, &executedOrder.IsWorking, &LastQueryTime)

		if err != nil {
			level.Error(logger).Log("getOrderList - Scan Fail. Err=", err)
			continue
		}

		if transactTime.Valid {
			executedOrder.Time = transactTime.Time
		}
		if LastQueryTime.Valid {
			orderData.LastQueryTime = LastQueryTime.Time
		}

		//fmt.Println("getOrderList - got OrderData:", orderData)

		// if already in open list
		for _, existing := range OpenOrderList {
			if existing.executedOrder.ClientOrderID == executedOrder.ClientOrderID {
				fmt.Println("warning! duplicate order id found. ClientOrderID=", executedOrder.ClientOrderID)
				continue rowLoopOpenOrder
			}
		}

		if !orderData.IsDone {
			OpenOrderList = append(OpenOrderList, orderData)
		}
	}

	if err := rows.Err(); err != nil {
		level.Error(logger).Log("getOrderList - rows.Err=", err)
		panic(err.Error())
	}

	//
	//fmt.Println("getOrderList - return", len(OpenOrderList), "orders. IsDone=", isDone)

	return OpenOrderList
}

/*
 * Get recent orders from local database for one asset.
 * Return: OrderList and Whether projectID exist in this list.
 */
func getRecentOrderList(symbol string, interval binance.Interval, projectID int64) ([]OrderData, bool) {

	orderList := make([]OrderData, 0)
	projectIdExist := false

	query := "select * from order_list where Symbol='" + symbol +
		"' and LastQueryTime > DATE_SUB(NOW(), INTERVAL "

	switch interval {
	case binance.ThreeDays:
		query += "3 DAY)"
	case binance.Week:
		query += "1 WEEK)"
	case binance.Month:
		query += "1 MONTH)"
	default:
		query += "1 DAY)"
	}

	query += " and ProjectID=-1 or " + fmt.Sprint(projectID) + " order by id desc"

	rows, err := DBCon.Query(query)

	if err != nil {
		level.Error(logger).Log("getOrderList - DB.Query Fail. Err=", err)
		panic(err.Error())
	}
	defer rows.Close()

	for rows.Next() {

		var transactTime NullTime
		var LastQueryTime NullTime

		orderData := OrderData{}
		executedOrder := &orderData.executedOrder

		err := rows.Scan(&orderData.id, &orderData.ProjectID, &orderData.IsDone,
			&executedOrder.Symbol, &executedOrder.OrderID, &executedOrder.ClientOrderID,
			&executedOrder.Price, &executedOrder.OrigQty, &executedOrder.ExecutedQty,
			&executedOrder.Status, &executedOrder.TimeInForce, &executedOrder.Type,
			&executedOrder.Side, &executedOrder.StopPrice, &executedOrder.IcebergQty,
			&transactTime, &executedOrder.IsWorking, &LastQueryTime)

		if err != nil {
			level.Error(logger).Log("getRecentOrderList - Scan Fail. Err=", err)
			continue
		}

		if transactTime.Valid {
			executedOrder.Time = transactTime.Time
		}
		if LastQueryTime.Valid {
			orderData.LastQueryTime = LastQueryTime.Time
		}

		//fmt.Println("getRecentOrderList - got OrderData:", orderData)

		if orderData.ProjectID == projectID{
			projectIdExist = true
		}

		orderList = append(orderList, orderData)
	}

	if err := rows.Err(); err != nil {
		level.Error(logger).Log("getRecentOrderList - rows.Err=", err)
		panic(err.Error())
	}

	//
	//fmt.Println("getRecentOrderList - return", len(OpenOrderList), "orders. IsDone=", isDone)

	return orderList, projectIdExist
}

