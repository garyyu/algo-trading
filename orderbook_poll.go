package main

import (
	"github.com/go-kit/kit/log/level"
	"time"
	"fmt"
	"strconv"
	"strings"
	"bitbucket.org/garyyu/algo-trading/go-binance"
)

type ObType int

const (
	Bid ObType = iota
	Ask
	Na
)
type OBData struct {
	id						 int		`json:"id"`
	LastUpdateID			 int		`json:"LastUpdateID"`
	Symbol            		 string 	`json:"Symbol"`
	Type 					 ObType		`json:"Type"`
	Price					 float64	`json:"Price"`
	Quantity				 float64	`json:"Quantity"`
	Time                	 time.Time	`json:"Time"`
}

/*
 *  Main Routine for OrderBook
 */
func OrderBookRoutine(){

	//time.Sleep(15 * time.Second)

	fmt.Printf("OrdBkTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	lowerIntervalCount := 0

	// start a goroutine to get realtime ROI analysis in 1 min interval
	ticker := orderbookTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			ticker.Stop()

			tickerCount += 1
			fmt.Printf("OrdBkTick: \t\t%s\t%d\n", tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)

			// for alive project list symbols, polling in quick interval: 3s
			ProjectMutex.RLock()
			for _, project := range ActiveProjectList {
				getOrderBook(project.Symbol, 10)
			}
			ProjectMutex.RUnlock()

			UpdateProjectNowPrice()

			// hunt list also
			huntList := getHuntList()
			for _, hunt := range *huntList {
				getOrderBook(hunt.Symbol, 10)
			}

			// for full price list, polling in lower interval: 30s
			lowerIntervalCount += 1
			if lowerIntervalCount >= 10 {
				for _, symbol := range LivelySymbolList {
					getOrderBook(symbol, 10)
				}
			}

			// Update the ticker
			ticker = orderbookTicker()

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - OrderBookRoutine")
}


func orderbookTicker() *time.Ticker {

	now := time.Now()
	return time.NewTicker(
		time.Second * time.Duration(3) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}


func getOrderBook(symbol string, limit int) (int,int){

	var bidsNum = 0
	var asksNum = 0
	var retry = 0
	for {
		retry += 1

		ob, err := binanceSrv.OrderBook(binance.OrderBookRequest{
			Symbol:   symbol,
			Limit:    limit,
		})
		if err != nil {
			level.Error(logger).Log("getOrderBook.Symbol", symbol, "Err", err, "Retry", retry-1)
			if retry >= 10 {
				break
			}

			switch retry {
			case 1:
				time.Sleep(1 * time.Second)
			case 2:
				time.Sleep(1 * time.Second)
			case 3:
				time.Sleep(3 * time.Second)
			case 4:
				time.Sleep(3 * time.Second)
			default:
				time.Sleep(5 * time.Second)
			}
			continue
		}

		if getLastUpdateId(symbol, ob.LastUpdateID) < 0 {
			saveOrderBook(symbol, "binance.com", ob)
			bidsNum = len(ob.Bids)
			asksNum = len(ob.Asks)
		}
		//else{
		//	fmt.Println("getOrderBook: got same LastUpdateID - ", ob.LastUpdateID)
		//}

		break
	}

	return bidsNum,asksNum
}

func saveOrderBook(symbol string, exchangeName string, ob *binance.OrderBook) error {

	if exchangeName != "binance.com"{
		level.Error(logger).Log("saveOrderBook.TODO", exchangeName)
		return nil
	}

	if len(ob.Asks)==0 && len(ob.Bids)==0 {
		level.Error(logger).Log("saveOrderBook.Empty", ob)
		return nil
	}

	sqlStr := "INSERT INTO ob_binance (LastUpdateID, Symbol, Type, Price, Quantity, Time) VALUES "
	var vals []interface{}

	// Bids
	for i:=len(ob.Bids)-1; i>=0; i-- {
		row := ob.Bids[i]
		sqlStr += "(?, ?, ?, ?, ?, NOW()),"
		vals = append(vals, ob.LastUpdateID, symbol, "Bid", row.Price, row.Quantity)
	}
	//trim the last ,
	sqlStr = strings.TrimSuffix(sqlStr, ",")

	stmt, err := DBCon.Prepare(sqlStr)
	if err != nil {
		level.Error(logger).Log("DBCon.Prepare", err, "sqlStr.len", len(sqlStr))
		return err
	}
	_, err2 := stmt.Exec(vals...)
	if err2 != nil {
		level.Error(logger).Log("DBCon.Exec", err2)
		return err2
	}


	// Asks
	vals = nil
	sqlStr = "INSERT INTO ob_binance (LastUpdateID, Symbol, Type, Price, Quantity, Time) VALUES "
	for _, row := range ob.Asks {
		sqlStr += "(?, ?, ?, ?, ?, NOW()),"
		vals = append(vals, ob.LastUpdateID, symbol, "Ask", row.Price, row.Quantity)
	}
	//trim the last ,
	sqlStr = strings.TrimSuffix(sqlStr, ",")

	stmt, err3 := DBCon.Prepare(sqlStr)
	if err3 != nil {
		level.Error(logger).Log("DBCon.Prepare", err3, "sqlStr.len", len(sqlStr))
		return err3
	}
	_, err4 := stmt.Exec(vals...)
	if err4 != nil {
		level.Error(logger).Log("DBCon.Exec", err4)
		return err4
	}

	//id, _ := res.LastInsertId()
	return nil
}


func getLastUpdateId(symbol string, LastUpdateID int) int{

	rows, err := DBCon.Query("select id from ob_binance where Symbol='" +
		symbol + "' and LastUpdateID='" + strconv.Itoa(LastUpdateID) + "' limit 1")

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	var id int = -1	// if not found, rows is empty.
	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			level.Error(logger).Log("getLastUpdateId.err", err)
			id = -1
		}
	}
	return id
}


func getHighestBid(symbol string) OBData{

	rows, err := DBCon.Query("select * from ob_binance where Symbol='" +
		symbol + "' and Type='Bid' order by id desc limit 1")

	if err != nil {
		level.Error(logger).Log("getHighestBid - DBCon.Query fail! Err:", err)
		panic(err.Error())
	}
	defer rows.Close()

	obData := OBData{id: -1}
	var obType string
	for rows.Next() {
		err := rows.Scan(&obData.id, &obData.LastUpdateID, &obData.Symbol, &obType,
			&obData.Price, &obData.Quantity, &obData.Time)
		if err != nil {
			level.Error(logger).Log("getHighestBid.err", err)
		}

		switch obType {
		case "Ask":	obData.Type = Ask
		case "Bid": obData.Type = Bid
		default: 	obData.Type = Na
		}
		break
	}
	return obData
}

/*
 * Insert Project data into Database
 */
func UpdateProjectNowPrice(){

	//fmt.Println("UpdateProjectNowPrice func enter. aliveProjects=", len(AliveProjectList))

	ProjectMutex.Lock()
	defer ProjectMutex.Unlock()

	for _, project := range ActiveProjectList {

		highestBid := getHighestBid(project.Symbol)
		if highestBid.Time.Add(time.Second * 5).Before(time.Now()) {
			continue
		}
		if highestBid.Symbol != project.Symbol || highestBid.Quantity <= 0{
			level.Error(logger).Log("Symbol", project.Symbol, "highestBid data exception!", highestBid)
			continue
		}

		// Update AliveProjectList NowPrice data
		project.NowPrice = highestBid.Price
		if project.InitialPrice>0 {
			project.RoiS = project.NowPrice/project.InitialPrice - 1.0
		}

		query := `UPDATE project_list SET NowPrice=?, RoiS=? WHERE id=?`

		res, err := DBCon.Exec(query,
			project.NowPrice,
			project.RoiS,
			project.id,
		)

		if err != nil {
			level.Error(logger).Log("DBCon.Exec", err)
			continue
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected <= 0 {
			level.Error(logger).Log("UpdateProjectNowPrice - Fail for project id:",
				project.id, "Symbol:", project.Symbol)
		}
	}

	//fmt.Println("UpdateProjectNowPrice func left")
}


