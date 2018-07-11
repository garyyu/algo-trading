package main

import (
	"fmt"
	"time"
	"bitbucket.org/garyyu/go-binance"
)

/*
 * KLines Data Updating. 5Min Lines
 */
func Ohlc5MinRoutine() {

	interval := binance.FiveMinutes

	totalQueryRet := 0
	totalQueryNewRet := 0

	fmt.Printf("%s KlineTick Start: \t%s\n\n", string(interval),
		time.Now().Format("2006-01-02 15:04:05.004005683"))

	// then we start a goroutine to get realtime data in intervals
	ticker := minuteTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			ticker.Stop()

			tickerCount += 1
			fmt.Printf("%s KlineTick: \t\t%s\t%d\n", string(interval),
				tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)
			_, min, _ := tick.Clock()
			if min % 5 == 0 {
				time.Sleep(5 * time.Second) // wait 5 seconds to ensure server data ready.
			}

			csvPollList := getCsvPollConf(interval)

			totalQueryRet = 0
			totalQueryNewRet = 0
			for _,symbol := range SymbolList {
				rowsNum,rowsNewNum := getKlinesData(symbol, 2, interval)
				totalQueryRet += rowsNum
				totalQueryNewRet += rowsNewNum

				// if need polling to csv file
				if _,ok := csvPollList[symbol]; ok{
					csvPolling(symbol, interval)
				}
			}
			fmt.Println("Poll", interval, "Klines from Binance -", len(SymbolList), "symbols.",
				"average:", float32(totalQueryRet)/float32(len(SymbolList)),
				"average new:", float32(totalQueryNewRet)/float32(len(SymbolList)),
				"\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

			// Update the ticker
			ticker = minuteTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - updateOhlc", string(interval))
}

func minuteTicker() *time.Ticker {
	// Return new ticker that triggers on the minute
	now := time.Now()
	return time.NewTicker(
		time.Second * time.Duration(60-now.Second()) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}
