package main

import (
	"context"
	"fmt"
	"os"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/garyyu/go-binance"
	"os/signal"
	"time"
)

var (
	DBCon 				*sql.DB				// the connection handle for the database
	binanceSrv 			binance.Binance
	routinesExitChan	chan bool
	logger 				log.Logger
)

func main() {

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger, "time", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	var err error
	DBCon, err = sql.Open("mysql",
		os.Getenv("BINANCE_DB_USER")+":"+
		os.Getenv("BINANCE_DB_PWD")+
		"@/binance?parseTime=true")
	if err != nil {
		panic(err.Error())
	}
	defer DBCon.Close()

	hmacSigner := &binance.HmacSigner{
		Key: []byte(os.Getenv("BINANCE_SECRET")),
	}
	ctx, cancelCtx := context.WithCancel(context.Background())
	// use second return value for cancelling request
	binanceService := binance.NewAPIService(
		"https://www.binance.com",
		os.Getenv("BINANCE_APIKEY"),
		hmacSigner,
		logger,
		ctx,
	)
	binanceSrv = binance.NewBinance(binanceService)

	routinesExitChan = make(chan bool)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go updateOhlc()

	fmt.Println("main is runing and waiting for interrupt")
	<-interrupt
	fmt.Println("Interrupt received. Canceling context ...")

	// notify all routines exit.
	close(routinesExitChan)
	time.Sleep(1 * time.Second)		// wait 1 seconds for routines exit

	cancelCtx()
	fmt.Println("waiting for signal")

	fmt.Println("main exited.")
	return
}

func updateOhlc() {

	var symbol string

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

		if count == 0 {
			getKlinesData(symbol, 1000)		// 1000*5 = 5000(mins) = 83 (hours) ~= 3.5 (days)
			time.Sleep(10 * time.Millisecond)	// avoid being baned by server
		//}else{
		//	getKlinesData(symbol, 100)		// 100*5 = 500(mins) = 8.3 (hours)
		//	time.Sleep(10 * time.Millisecond)	// avoid being baned by server
		}else{
			getKlinesData(symbol, 12)		// 12*5 = 60(mins) = 1 (hour)
			time.Sleep(10 * time.Millisecond)	// avoid being baned by server
		}

		select {
		case _ = <- routinesExitChan:
			break initialDataLoop

		default:
			time.Sleep(10 * time.Millisecond)
		}

	}

	fmt.Printf("\nTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

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
			fmt.Printf("Tick: \t\t%s\t%d\n", time.Now().Format("2006-01-02 15:04:05.004005683"), tickerCount)
			_, min, _ := tick.Clock()
			if min % 5 == 0 {
				time.Sleep(5 * time.Second) // wait 5 seconds to ensure server data ready.
			}
			for _,symbol = range SymbolList {
				getKlinesData(symbol, 2)
			}

			// Update the ticker
			ticker = minuteTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - updateOhlc")
}

func getKlineId(symbol string, openTime time.Time) (int64,time.Time){

	rows, err := DBCon.Query("select id,insertTime from ohlc5min where Symbol='" + symbol + "' and OpenTime='" +
					openTime.Format("2006-01-02 15:04:05") + " limit 1'")

	//level.Debug(logger).Log("getKlinesId.Query", openTime.Format("2006-01-02 15:04:05"))

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	var id int64 = -1	// if not found, rows is empty.
	var insertTime time.Time
	for rows.Next() {
		err := rows.Scan(&id, &insertTime)
		if err != nil {
			level.Error(logger).Log("getKlineId.err", err)
			id = -1
		}
	}
	//fmt.Println("getKlinesId() for", symbol, "at time",
	//	openTime.Format("2006-01-02 15:04:05"), " id=", id,
	//	"insertTime=", insertTime.Format("2006-01-02 15:04:05"))
	return id,insertTime
}

func getKlinesData(symbol string, limit int){

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

		fmt.Printf("%s - %s received %d klines\n",
			time.Now().Format("2006-01-02 15:04:05.004005683"), symbol, len(kl))
		for _, v := range kl {
			//fmt.Printf("%d %v\n", i, v)
			id, insertTime := getKlineId(symbol, v.OpenTime)
			if id < 0 {
				OhlcCreate(symbol, "binance.com", *v)
			} else {
				// update it
				OhlcUpdate(id, insertTime, symbol, "binance.com", *v)
			}
		}

		break
	}
}

func minuteTicker() *time.Ticker {
	// Return new ticker that triggers on the minute
	now := time.Now()
	return time.NewTicker(
		time.Second * time.Duration(60-now.Second()) -
			time.Duration(now.Nanosecond())*time.Nanosecond)
}
