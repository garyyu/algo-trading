package main

import (
	"github.com/go-kit/kit/log/level"
	"fmt"
	"bitbucket.org/garyyu/go-binance"
	"time"
	"math/rand"
)

type TradeData struct {
	id				int64	`json:"id"`
	ProjectID		int64	`json:"ProjectID"`
	Symbol          string 	`json:"Symbol"`
	TradeID         int64	`json:"TradeID"`
	Price           float64	`json:"Price"`
	Qty             float64	`json:"Qty"`
	Commission      float64	`json:"Commission"`
	CommissionAsset string	`json:"CommissionAsset"`
	Time            time.Time	`json:"Time"`
	IsBuyer         bool	`json:"IsBuyer"`
	IsMaker         bool	`json:"IsMaker"`
	IsBestMatch     bool	`json:"IsBestMatch"`
	InsertTime		time.Time	`json:"InsertTime"`
}

/*
 * Expensive API: Weight=5
 */
func QueryAccount() {

	accountInfo, err := binanceSrv.Account(binance.AccountRequest{
		RecvWindow: 5 * time.Second,
		Timestamp:  time.Now(),
	})
	if err != nil {
		level.Error(logger).Log("QueryAccount - fail! Err=", err)
		return
	}

	lookForNew:
	for _, balance := range accountInfo.Balances {

		if balance.Asset == "BTC" || balance.Asset == "ETH" {
			continue
		}

		symbol := balance.Asset + "BTC"

		// get latest price
		highestBid := getHighestBid(symbol)
		if highestBid.Time.Add(time.Second * 60).Before(time.Now()) {
			fmt.Println("Warning! QueryAccount - getHighestBid got old data. fail to manage its project", symbol)
			continue
		}

		// ignore trivial balance
		if highestBid.Price * balance.Free < 5 * MinOrderTotal {
			continue
		}

		for _, knownProject := range AliveProjectList {

			// Existing Known Project?
			if knownProject.Symbol == balance.Asset+"BTC" {

				// Update the Balance to the local database
				if knownProject.BalanceBase != balance.Free {
					knownProject.BalanceBase = balance.Free
					if !UpdateProjectBalanceBase(&knownProject){
						fmt.Println("QueryAccount - Update Project BalanceBase fail. project:", knownProject)
					}
				}

				continue lookForNew
			}
		}

		// Must Be a New Project!
		ProjectImport(balance)
	}

	// reverse looking for Close Project
	reverseLooking:
	for _, aliveProject := range AliveProjectList {

		for _, balance := range accountInfo.Balances {
			if aliveProject.Symbol == balance.Asset+"BTC" {
				continue reverseLooking
			}
		}

		// Must Be a Close Project! (i.e. already sold asset.)
		ProjectClose(&aliveProject)
	}

}

/*
 * Expensive API: Weight=5
 * 		Only call it when we don't know the OrderID, for example when import project.
 */
func QueryMyTrades(){

	ProjectMutex.Lock()
	defer ProjectMutex.Unlock()

	for _, aliveProject := range AliveProjectList {

		if aliveProject.IsClosed {
			continue
		}

		myTrades, err := binanceSrv.MyTrades(binance.MyTradesRequest{
			Symbol:     aliveProject.Symbol,
			RecvWindow: 5 * time.Second,
			Timestamp:  time.Now(),
		})
		if err != nil {
			level.Error(logger).Log("QueryMyTrades - fail! Symbol=", aliveProject.Symbol)
			return
		}

		for _, trade := range myTrades {

			if GetTradeId(trade.ID)<=0 {
				if InsertTrade(aliveProject.Symbol, trade)<0 {
					level.Error(logger).Log("QueryMyTrades - InsertTrade fail! trade:", trade)
				}
			}
		}

		// Get recent trades list in this asset, with the order of latest first.
		tradeList := getRecentTradeList(aliveProject.Symbol, binance.Day)
		amount := 0.0
		invest := 0.0

		tradesNum := 0
		for _,trade := range tradeList {
			tradesNum += 1

			if trade.IsBuyer {
				amount += trade.Qty
				invest += trade.Qty * trade.Price
			}else{
				amount -= trade.Qty
				invest -= trade.Qty * trade.Price
			}

			if amount == aliveProject.InitialAmount {
				// Finally! We find the trade(s) where this asset balance came from.
				aliveProject.InitialBalance = invest
				aliveProject.InitialPrice = invest / amount		// average price if multiple trades
				break
			}
		}

		// We find it? Let's put the ProjectID into all these trades
		if aliveProject.InitialBalance>0 {

			for i:=0; i<tradesNum; i++{
				trade := tradeList[i]
				trade.ProjectID = aliveProject.id

				if !UpdateTradeProjectID(&trade){
					fmt.Println("QueryMyTrades - UpdateTradeProjectID Failed. trade:", trade)
				}
			}

			if !UpdateProjectInitialBalance(&aliveProject){
				fmt.Println("QueryMyTrades - Warning! Update Project InitialBalance into database Fail. aliveProject:",
					aliveProject)
			}
		}else{
			fmt.Println("QueryMyTrades - Warning! new project for asset", aliveProject.Symbol,
				"not found in my trades history! Project can't be managed.")
			continue
		}

	}

}

/*
 * Insert Trade data into Database
 */
func InsertTrade(symbol string, trade *binance.Trade) int64{

	query := `INSERT INTO trade_list (
				Symbol, TradeID, Price, Qty, Commission, 
				CommissionAsset, Time, IsBuyer, IsMaker, IsBestMatch, InsertTime
			  ) VALUES (?,?,?,?,?,?,?,?,?,?,NOW())`

	res, err := DBCon.Exec(query,
					symbol,
					trade.ID,
					trade.Price,
					trade.Qty,
					trade.Commission,
					trade.CommissionAsset,
					trade.Time,
					trade.IsBuyer,
					trade.IsMaker,
					trade.IsBestMatch,
	)

	if err != nil {
		level.Error(logger).Log("InsertTrade - DBCon.Exec", err)
		return -1
	}

	id, _ := res.LastInsertId()
	return id
}


/*
 * Used for detect if trade exist in local database
 */
func GetTradeId(TradeID int64) int64 {

	row := DBCon.QueryRow("SELECT id FROM trade_list WHERE TradeID=?", TradeID)

	var id int64 = -1
	err := row.Scan(&id)
	if err != nil {
		level.Error(logger).Log("GetTradeId - Scan Err:", err)
	}

	return id
}


/*
 * Update Trade ProjectID into Database
 */
func UpdateTradeProjectID(tradeData *TradeData) bool{

	query := `UPDATE trade_list SET ProjectID=? WHERE id=?`

	res, err := DBCon.Exec(query,
		tradeData.ProjectID,
		tradeData.id,
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
 * Get recent trades from local database for one asset
 */
func getRecentTradeList(symbol string, interval binance.Interval) []TradeData{

	tradeList := make([]TradeData, 0)

	query := "select * from trade_list where Symbol='" + symbol +
		"' and InsertTime > DATE_SUB(NOW(), INTERVAL "

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

	query += " order by Time desc"

	rows, err := DBCon.Query(query)

	if err != nil {
		level.Error(logger).Log("getRecentTradeList - DBCon.Exec", err)
		panic(err.Error())
	}
	defer rows.Close()

	for rows.Next() {

		tradeData := TradeData{}

		err := rows.Scan(&tradeData.ProjectID,
			&tradeData.id, &tradeData.Symbol, &tradeData.TradeID,
			&tradeData.Price, &tradeData.Qty, &tradeData.Commission,
			&tradeData.CommissionAsset, &tradeData.Time, &tradeData.IsBuyer,
			&tradeData.IsMaker, &tradeData.IsBestMatch, &tradeData.InsertTime)

		if err != nil {
			level.Error(logger).Log("getRecentTradeList - Scan Err:", err)
			continue
		}

		tradeList = append(tradeList, tradeData)
	}

	return tradeList
}

func ProjectClose(project *ProjectData){

	project.IsClosed = true

	if !UpdateProjectClose(project) {
		level.Error(logger).Log("ProjectClose - database update fail! project=", project)
	}else{
		fmt.Println("ProjectClose - Done. Project Info:", project)
	}
}

func ProjectImport(balance *binance.Balance){

	ClientOrderID := time.Now().Format("20060102150405") + fmt.Sprintf("%04d",rand.Intn(9999))

	symbol := balance.Asset + "BTC"

	// create new project with hunt.Symbol
	NewProject := ProjectData{
		id:-1,
		Symbol:symbol,
		ClientOrderID: ClientOrderID,
		InitialAmount: balance.Free,
		OrderStatus: string(binance.StatusNew),
	}

	// save into database
	id := InsertProject(&NewProject)
	if id<0 {
		level.Error(logger).Log("Error! InsertProject fail. NewProject=", NewProject)
		return
	}
	NewProject.id = id
}
