
package main

import (
	"time"
	."github.com/garyyu/go-binance"
	"github.com/go-kit/kit/log/level"
)

type OhlcDbTbl struct {
	Id              		 int64 		`json:"id"`
	Symbol            		 string 	`json:"Symbol"`
	OpenTime                 time.Time	`json:"OpenTime"`
	Open                     float64	`json:"Open"`
	High                     float64	`json:"High"`
	Low                      float64	`json:"Low"`
	Close                    float64	`json:"Close"`
	Volume                   float64	`json:"Volume"`
	CloseTime                time.Time	`json:"CloseTime"`
	QuoteAssetVolume         float64	`json:"QuoteAssetVolume"`
	NumberOfTrades           int		`json:"NumberOfTrades"`
	TakerBuyBaseAssetVolume  float64	`json:"TakerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume float64	`json:"TakerBuyQuoteAssetVolume"`
	exchangeName			 string		`json:"exchangeName"`
	insertTime				 time.Time	`json:"insertTime"`
	updateTime				 time.Time	`json:"updateTime"`
}


func OhlcCreate(symbol string, exchangeName string, kline Kline) (ohldDblTbl OhlcDbTbl, err error) {

	query := `INSERT INTO ohlc5min (
		Symbol, OpenTime, Open, High, Low, Close, Volume, CloseTime,
		QuoteAssetVolume, NumberOfTrades, TakerBuyBaseAssetVolume, TakerBuyQuoteAssetVolume,
		exchangeName, insertTime
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	res, err := DBCon.Exec(query,
		symbol,
		kline.OpenTime,
		kline.Open,
		kline.High,
		kline.Low,
		kline.Close,
		kline.Volume,
		kline.CloseTime,
		kline.QuoteAssetVolume,
		kline.NumberOfTrades,
		kline.TakerBuyBaseAssetVolume,
		kline.TakerBuyQuoteAssetVolume,
		exchangeName,
		time.Now().UTC(),
	)
	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return OhlcDbTbl{}, err
	}

	id, _ := res.LastInsertId()
	return OhlcDbTbl{
		id,
		symbol,
		kline.OpenTime,
		kline.Open,
		kline.High,
		kline.Low,
		kline.Close,
		kline.Volume,
		kline.CloseTime,
		kline.QuoteAssetVolume,
		kline.NumberOfTrades,
		kline.TakerBuyBaseAssetVolume,
		kline.TakerBuyQuoteAssetVolume,
		exchangeName,
		time.Now(),
		time.Now(),
	}, nil
}
