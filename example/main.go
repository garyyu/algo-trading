package main

import (
	"context"
	"fmt"
	"os"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/garyyu/go-binance"
	"os/signal"
)

func main() {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger, "time", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	hmacSigner := &binance.HmacSigner{
		Key: []byte("2BE6tDP7ljWavOS8U36wSS5GEAgsff7Dlo0zrk0BP7OJdX9MUbiQVwgVTIdUutWI"),
	}
	ctx, cancelCtx := context.WithCancel(context.Background())
	// use second return value for cancelling request
	binanceService := binance.NewAPIService(
		"https://www.binance.com",
		"lJUy9vDFZ1fUUwphvde5oQuCmsEfzoKGaKnbJ3oUiMVQQasj6AkxlJe6Zf8hDcyv",
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
		Limit: 1000,
		//StartTime: 1522965600,
		//EndTime: 1522965600-3600*1000,
	})
	if err != nil {
		panic(err)
	}

	for i, v := range kl {
		fmt.Printf("%d %v\n", i, v)
	}

	fmt.Println("waiting for interrupt")
	<-interrupt
	fmt.Println("canceling context")
	cancelCtx()
	fmt.Println("waiting for signal")
	return
}
