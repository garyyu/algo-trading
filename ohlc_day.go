package main

import (
	"time"
	"fmt"
	"bitbucket.org/garyyu/algo-trading/go-binance"
)

/*
 * Daily KLines
 */
func DailyOhlcRoutine() {

	interval := binance.Day

	totalQueryRet := 0
	totalQueryNewRet := 0

	fmt.Printf("%s KlineTick Start: \t%s\n\n", string(interval),
		time.Now().Format("2006-01-02 15:04:05.004005683"))

	// then we start a goroutine to get realtime data in intervals
	ticker := dayTicker()
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
			for _,symbol := range AllSymbolList {
				rowsNum,rowsNewNum := getKlinesData(symbol, 2, interval)
				totalQueryRet += rowsNum
				totalQueryNewRet += rowsNewNum

				// if need polling to csv file
				if _,ok := csvPollList[symbol]; ok{
					csvPolling(symbol, interval)
				}
			}
			//fmt.Printf("Poll %s Klines from Binance - %d symbols. average: %.1f, average new: %.1f\t\t%s\n",
			//	interval, len(SymbolList),
			//	float32(totalQueryRet)/float32(len(SymbolList)),
			//	float32(totalQueryNewRet)/float32(len(SymbolList)),
			//	time.Now().Format("2006-01-02 15:04:05.004005683"))

			// Update the ticker
			ticker = dayTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - updateOhlc", string(interval))
}

func dayTicker() *time.Ticker {

	now := time.Now()
	return time.NewTicker(
		time.Minute * time.Duration(60-now.Minute()) -
		time.Second * time.Duration(now.Second()) -
		time.Nanosecond * time.Duration(now.Nanosecond()))
}
