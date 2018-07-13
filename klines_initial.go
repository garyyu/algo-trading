package main

import (
	"fmt"
	"time"
	"bitbucket.org/garyyu/algo-trading/go-binance"
	"math"
)

/*
 * Initial KLines data downloading from Binance.
 *		- Run only once when program start-up
 *		- If local database is empty (for one Symbol), download 1000 (max allowed by Binance) data;
 *		- Otherwise, download number according to required missed time.
 */
func InitialKlines(interval binance.Interval){

	fmt.Println("\nInitialize", string(interval), "KLines from Binance ...\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

	var totalQueryRet = 0
	var totalQueryNewRet = 0

initialDataLoop:
	for i,symbol := range SymbolList {

		select {
		case _ = <- routinesExitChan:
			break initialDataLoop

		default:
			time.Sleep(10 * time.Millisecond)
		}

		var rowsNum = 0
		var rowsNewNum = 0

		fmt.Printf("\b\b\b\b\b\b%.1f%% ", float64(i+1)*100.0/float64(len(SymbolList)))

		limit := getLimit(symbol, interval)

		rowsNum,rowsNewNum = getKlinesData(symbol, limit, interval)
		time.Sleep(10 * time.Millisecond)	// avoid being baned by server

		totalQueryRet += rowsNum
		totalQueryNewRet += rowsNewNum

	}
	fmt.Printf("\n%s KLines Initialization Done. - %d symbols. average: %.2f, average new: %.2f\n",
		string(interval), len(SymbolList),
		float32(totalQueryRet)/float32(len(SymbolList)),
		float32(totalQueryNewRet)/float32(len(SymbolList)))
}

func getLimit(symbol string, interval binance.Interval) int{

	const MAXLIMIT = 1000			// 1000*5 = 5000(Min) = 83 (hours) ~= 3.5 (days)
	limit := 12						//   12*5 = 60(Min)   = 1 Hour

	OpenTime := GetOpenTime(symbol, interval)
	if OpenTime.IsZero() {
		return MAXLIMIT
	}

	duration := time.Since(OpenTime)

	switch interval{
	case binance.FiveMinutes:
		limit = 3 + int( math.Max(duration.Minutes() / 5.0, 0) )
	case binance.Hour:
		limit = 3 + int( math.Max(duration.Hours(), 0) )
	case binance.Day:
		limit = 3 + int( math.Max(duration.Hours() / 24.0, 0) )
	default:
		return limit
	}
	//
	//fmt.Printf("getLimit - %s:%s limit=%d\n",symbol, interval, limit)

	if limit>MAXLIMIT{
		limit = MAXLIMIT
	}

	return limit
}

