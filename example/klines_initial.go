package main

import (
	"bitbucket.org/garyyu/go-binance"
	"fmt"
	"time"
)

/*
 * Initial KLines data downloading from Binance.
 *		- Run only once when program start-up
 *		- If local database is empty (for one Symbol), download 1000 (max allowed by Binance) data;
 *		- Otherwise, download number according to required missed time.
 */
func InitialKlines(interval binance.Interval){

	var symbol string

	fmt.Println("Initialize ", string(interval), "KLines from Binance ...\t\t\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

	var totalQueryRet = 0
	var totalQueryNewRet = 0

initialDataLoop:
	for _,symbol = range SymbolList {

		//query database if it's a new import symbol.
		rows, err := DBCon.Query("select count(id) as count from ohlc_" + string(interval) +
			" where Symbol='" + symbol + "'")
		if err != nil {
			panic(err.Error())
		}

		var count int // we "scan" the result in here
		for rows.Next() {
			err := rows.Scan(&count)
			if err != nil {
				count = 0
			}
		}
		rows.Close()

		var rowsNum = 0
		var rowsNewNum = 0
		if count == 0 {
			rowsNum,rowsNewNum = getKlinesData(symbol, 1000,
				interval)											// 1000*5 = 5000(Mins) = 83 (hours) ~= 3.5 (days)
			time.Sleep(10 * time.Millisecond)						// avoid being baned by server
		}else{
			rowsNum,rowsNewNum = getKlinesData(symbol, 12,
				interval)											// 12*5 = 60(Mins) = 1 (hour)
			time.Sleep(10 * time.Millisecond)						// avoid being baned by server
		}
		totalQueryRet += rowsNum
		totalQueryNewRet += rowsNewNum

		select {
		case _ = <- routinesExitChan:
			break initialDataLoop

		default:
			time.Sleep(10 * time.Millisecond)
		}

	}
	fmt.Println(string(interval), "KLines Initialization Done - ", len(SymbolList), "symbols",
		" average KLines :", float32(totalQueryRet)/float32(len(SymbolList)),
		" average new KLines:", float32(totalQueryNewRet)/float32(len(SymbolList)))
}

