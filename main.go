package main

import (
	"context"
	"fmt"
	"os"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"os/signal"
	"time"
	"bitbucket.org/garyyu/algo-trading/go-binance"
	"sync"
)

var (
	DBCon 				*sql.DB				// the connection handle for the database
	binanceSrv 			binance.Binance
	routinesExitChan	chan bool
	logger 				log.Logger
)

func initialization(){

	ProjectTrackIni()
}

func main() {

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger, "time", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	if len(string(os.Getenv("BINANCE_DB_USER")))==0 {
		fmt.Println("before running it, please load env variables before using it. exited.")
		return
	}

	var err error
	DBCon, err = sql.Open("mysql",
		os.Getenv("BINANCE_DB_USER")+":"+
		os.Getenv("BINANCE_DB_PWD")+
		"@/binance?parseTime=true&interpolateParams=true")
	if err != nil {
		panic(err.Error())
	}
	defer DBCon.Close()

	// Configuring sql.DB for Better Performance
	DBCon.SetMaxOpenConns(128)
	DBCon.SetMaxIdleConns(128)
	DBCon.SetConnMaxLifetime(time.Second * 128)

	initialization()

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

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go startMainRoutines(wg)

	fmt.Println("main is runing and waiting for interrupt")
	<-interrupt
	fmt.Println("Interrupt received. Canceling context ...")

	// notify all routines exit.
	close(routinesExitChan)
	fmt.Println("please waiting for a few seconds for sub-routines exit...")
	wg.Wait()

	cancelCtx()

	fmt.Println("main exited.")
	return
}

func startMainRoutines(wg *sync.WaitGroup){
	defer wg.Done()

	//----- downloading latest K lines data from Binance server	-----//
	InitialKlines(binance.Day, true)
	InitialKlines(binance.Hour, true)

	// ignore boring symbols whose 24Hour Volume < 50 BTC
	const ignoreBoringSymbol = true
	const ignoreQuoteVolume = 50.0

	if ignoreBoringSymbol{

		LivelySymbolList = nil
		for _,symbol := range AllSymbolList {

			realCount,quoteVolume := GetLastVolume(symbol, binance.Hour, 24)
			//fmt.Printf("%s: Last %dH Volume=%f\n", symbol, realCount, quoteVolume)

			if realCount==1 || quoteVolume>=ignoreQuoteVolume {
				LivelySymbolList = append(LivelySymbolList, symbol)
			}
		}

		fmt.Printf("\nTotal Symbols=%d and %d of them ignored tracking because poor volume.\n",
			len(AllSymbolList), len(AllSymbolList)-len(LivelySymbolList))
	}

	InitialKlines(binance.FiveMinutes, false)

	// loading K lines into memory from local database
	InitLocalKlines(binance.FiveMinutes)

	//----- 					all routines 					-----//

	wg.Add(1)
	go OrderBookRoutine(wg)

	wg.Add(1)
	go Ohlc5MinRoutine(wg)

	wg.Add(1)
	go HourlyOhlcRoutine(wg)

	wg.Add(1)
	go DailyOhlcRoutine(wg)

	// now it's good time to start ROI analysis routine
	//go RoiRoutine(wg)

	wg.Add(1)
	go HotspotRoutine(wg)

	// also start project manager
	wg.Add(1)
	go ProjectRoutine(wg)

	//-----   repeat loading from database for latest K lines	-----//

	loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop

		default:
			Klines5mMutex.Lock()
			RefreshKlines(binance.FiveMinutes)
			Klines5mMutex.Unlock()
			time.Sleep(15 * time.Second)
		}
	}

	return
}
