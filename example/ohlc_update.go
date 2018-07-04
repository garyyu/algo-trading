package main

import (
	"github.com/go-kit/kit/log/level"
	"time"
	"fmt"
	"github.com/garyyu/go-binance"
)

func updateOhlc() {

	var symbol string

	fmt.Println("Initialize KLines from Binance ...\t\t\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

	var totalQueryRet = 0
	var totalQueryNewRet = 0

initialDataLoop:
	for _,symbol = range SymbolList {

		//query database if it's a new import symbol.
		rows, err := DBCon.Query("select count(id) as count from ohlc5min where Symbol='" + symbol + "'")
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		var count int // we "scan" the result in here
		for rows.Next() {
			err := rows.Scan(&count)
			if err != nil {
				count = 0
			}
		}
		rows.Close()
		//fmt.Println("The local db existing records :", count, " on symbol:", symbol)

		var rowsNum = 0
		var rowsNewNum = 0
		if count == 0 {
			rowsNum,rowsNewNum = getKlinesData(symbol, 1000)	// 1000*5 = 5000(mins) = 83 (hours) ~= 3.5 (days)
			time.Sleep(10 * time.Millisecond)						// avoid being baned by server
		}else{
			rowsNum,rowsNewNum = getKlinesData(symbol, 100)		// 100*5 = 500(mins) = 8.3 (hours)
			time.Sleep(10 * time.Millisecond)						// avoid being baned by server
		//}else{
		//	rowsNum,rowsNewNum = getKlinesData(symbol, 12)			// 12*5 = 60(mins) = 1 (hour)
		//	time.Sleep(10 * time.Millisecond)						// avoid being baned by server
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
	fmt.Println("KLines Initialization Done - ", len(SymbolList), "symbols",
		" average KLines :", float32(totalQueryRet)/float32(len(SymbolList)),
		" average new KLines:", float32(totalQueryNewRet)/float32(len(SymbolList)))

	totalQueryRet = 0
	totalQueryNewRet = 0

	fmt.Printf("\nKlineTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	// then we start a goroutine to get realtime data in intervals
	ticker := minuteTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			tickerCount += 1
			fmt.Printf("KlineTick: \t\t%s\t%d\n", time.Now().Format("2006-01-02 15:04:05.004005683"), tickerCount)
			_, min, _ := tick.Clock()
			if min % 5 == 0 {
				time.Sleep(5 * time.Second) // wait 5 seconds to ensure server data ready.
			}

			totalQueryRet = 0
			totalQueryNewRet = 0
			for _,symbol = range SymbolList {
				rowsNum,rowsNewNum := getKlinesData(symbol, 2)
				totalQueryRet += rowsNum
				totalQueryNewRet += rowsNewNum
			}
			fmt.Println("PollKlines from Binance - ", len(SymbolList), "symbols",
				" average KLines:", float32(totalQueryRet)/float32(len(SymbolList)),
				" average new KLines:", float32(totalQueryNewRet)/float32(len(SymbolList)),
				"\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

			// Update the ticker
			ticker = minuteTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - updateOhlc")
}

func getKlineId(symbol string, openTime time.Time) (int64,time.Time,int){

	rows, err := DBCon.Query("select id,insertTime,UpdateTimes from ohlc5min where Symbol='" + symbol + "' and OpenTime='" +
		openTime.Format("2006-01-02 15:04:05") + " limit 1'")

	//level.Debug(logger).Log("getKlinesId.Query", openTime.Format("2006-01-02 15:04:05"))

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	var id int64 = -1	// if not found, rows is empty.
	var insertTime time.Time
	var updateTimes = 0
	for rows.Next() {
		err := rows.Scan(&id, &insertTime, &updateTimes)
		if err != nil {
			level.Error(logger).Log("getKlineId.err", err)
			id = -1
		}
	}
	//fmt.Println("getKlinesId() for", symbol, "at time",
	//	openTime.Format("2006-01-02 15:04:05"), " id=", id,
	//	"insertTime=", insertTime.Format("2006-01-02 15:04:05"))
	return id,insertTime,updateTimes
}

func getKlinesData(symbol string, limit int) (int,int){

	var rowsNum = 0
	var rowsNewNum = 0
	var retry = 0
	for {
		retry += 1

		kl, err := binanceSrv.Klines(binance.KlinesRequest{
			Symbol:   symbol,
			Interval: binance.FiveMinutes,
			Limit:    limit,
		})
		if err != nil {
			level.Error(logger).Log("getKlinesData.Symbol", symbol, "Err", err, "Retry", retry-1)
			if retry >= 10 {
				break
			}
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		if limit > 2 {
			fmt.Printf("%s - %s received %d klines\n",
				time.Now().Format("2006-01-02 15:04:05.004005683"), symbol, len(kl))
		}
		for _, v := range kl {
			rowsNum += 1
			id,insertTime,updateTimes := getKlineId(symbol, v.OpenTime)
			if id < 0 {
				OhlcCreate(symbol, "binance.com", *v)
				rowsNewNum += 1
			} else {
				// update it
				OhlcUpdate(id, insertTime, symbol, "binance.com", *v, updateTimes)
			}
		}

		break
	}

	return rowsNum,rowsNewNum
}

func minuteTicker() *time.Ticker {
	// Return new ticker that triggers on the minute
	now := time.Now()
	return time.NewTicker(
		time.Second * time.Duration(60-now.Second()) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}
