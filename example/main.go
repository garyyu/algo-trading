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
)

var (
	// DBCon is the connection handle for the database
	DBCon *sql.DB
	logger log.Logger
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
	b := binance.NewBinance(binanceService)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	kl, err := b.Klines(binance.KlinesRequest{
		Symbol:   "KEYBTC",
		Interval: binance.FiveMinutes,
		Limit: 10,
		//StartTime: 1522965600,
		//EndTime: 1522965600-3600*1000,
	})
	if err != nil {
		panic(err)
	}

	for i, v := range kl {
		//fmt.Printf("%d %v\n", i, v)
		OhlcCreate("KEYBTC", "binance.com", *v)
	}

	fmt.Println("waiting for interrupt")
	<-interrupt
	fmt.Println("canceling context")
	cancelCtx()
	fmt.Println("waiting for signal")
	return
}
