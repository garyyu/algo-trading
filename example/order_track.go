package main

import (
	"github.com/go-kit/kit/log/level"
	"fmt"
	"bitbucket.org/garyyu/go-binance"
	"time"
)

type OrderData struct {
	id				 		 int64		`json:"id"`
	ProjectID				 int64		`json:"ProjectID"`
	IsDone					 bool		`json:"IsDone"`
	executedOrder			 binance.ExecutedOrder	`json:"executedOrder"`
	LastQueryTime            time.Time	`json:"LastQueryTime"`
}

func QueryOrders(){

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
		if executedOrder.IsWorking {
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

	for _, project := range AliveProjectList {

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

	if orderData==nil || len(orderData.executedOrder.ClientOrderID)==0 {
		level.Warn(logger).Log("InsertOrder - invalid orderData!", orderData)
		return -1
	}

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

	fmt.Println("getOrderList - return", len(OpenOrderList), "orders. IsDone=", isDone)
	return OpenOrderList
}

