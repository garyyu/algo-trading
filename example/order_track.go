package main

import (
	"github.com/go-kit/kit/log/level"
	"fmt"
	"github.com/garyyu/go-binance"
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

	openOrderList := getOpenOrderList()

	for _, openOrder := range openOrderList {

		if openOrder.IsDone {
			continue
		}

		executedOrder, err := binanceSrv.QueryOrder(binance.QueryOrderRequest{
			Symbol:     openOrder.executedOrder.Symbol,
			OrigClientOrderID: openOrder.executedOrder.ClientOrderID,
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
		if !executedOrder.IsWorking {
			openOrder.IsDone = true
		}

		UpdateOrder(&openOrder)

		fmt.Println("QueryOrders - Result:", executedOrder)
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
				ProjectID, Symbol, ClientOrderID, Price, OrigQty, Status, TimeInForce, Type, Side
			  ) VALUES (?,?,?,?,?,?,?,?,?)`

	res, err := DBCon.Exec(query,
		orderData.ProjectID,
		orderData.executedOrder.Symbol,
		orderData.executedOrder.ClientOrderID,
		orderData.executedOrder.Price,
		orderData.executedOrder.OrigQty,
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

	query := `UPDATE order_list SET OrderID=?, Price=?, OrigQty=?, ExecutedQty=?, Status=?, TimeInForce=?,
				Type=?, Side=?, StopPrice=?, IcebergQty=?, Time=?, LastQueryTime=NOW() WHERE ClientOrderID=?`

	res, err := DBCon.Exec(query,
		executedOrder.OrderID,
		executedOrder.Price,
		executedOrder.OrigQty,
		executedOrder.ExecutedQty,
		executedOrder.Status,
		executedOrder.TimeInForce,
		executedOrder.Type,
		executedOrder.Side,
		executedOrder.StopPrice,
		executedOrder.IcebergQty,
		executedOrder.Time,
		executedOrder.ClientOrderID,
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
 * 'Open' means in local database view. To get status, query the Binance server via API.
 */
func getOpenOrderList() []OrderData {

	fmt.Println("getOpenOrderList func enter")

	rows, err := DBCon.Query("SELECT * FROM order_list WHERE IsDone=0 LIMIT 10")

	if err != nil {
		level.Error(logger).Log("getOpenOrderList - DB.Query Fail. Err=", err)
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
			&executedOrder.Symbol, &executedOrder.ClientOrderID, &executedOrder.Price,
			&executedOrder.OrigQty, &executedOrder.ExecutedQty, &executedOrder.Status, &executedOrder.TimeInForce,
			&executedOrder.Type, &executedOrder.Side, &executedOrder.StopPrice, &executedOrder.IcebergQty,
			&transactTime, &executedOrder.IsWorking, &LastQueryTime)

		if err != nil {
			level.Error(logger).Log("getOpenOrderList - Scan Fail. Err=", err)
			continue
		}

		if transactTime.Valid {
			executedOrder.Time = transactTime.Time
		}
		if LastQueryTime.Valid {
			orderData.LastQueryTime = LastQueryTime.Time
		}

		fmt.Println("getOpenOrderList - got OrderData:", orderData)

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
		level.Error(logger).Log("getOpenOrderList - rows.Err=", err)
		panic(err.Error())
	}

	fmt.Println("getOpenOrderList - return", len(OpenOrderList), "orders")
	return OpenOrderList
}

