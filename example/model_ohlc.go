
package main

import (
	"time"
	."bitbucket.org/garyyu/go-binance"
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
	UpdateTimes				 int		`json:"UpdateTimes"`
}


func OhlcCreate(symbol string, exchangeName string, kline Kline) error {

	query := `INSERT INTO ohlc5min (
		Symbol, OpenTime, Open, High, Low, Close, Volume, CloseTime,
		QuoteAssetVolume, NumberOfTrades, TakerBuyBaseAssetVolume, TakerBuyQuoteAssetVolume,
		exchangeName, insertTime
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,NOW())`
	_, err := DBCon.Exec(query,
		symbol,
		kline.OpenTime.Format("2006-01-02 15:04:05"),
		kline.Open,
		kline.High,
		kline.Low,
		kline.Close,
		kline.Volume,
		kline.CloseTime.Format("2006-01-02 15:04:05"),
		kline.QuoteAssetVolume,
		kline.NumberOfTrades,
		kline.TakerBuyBaseAssetVolume,
		kline.TakerBuyQuoteAssetVolume,
		exchangeName,
	)
	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return err
	}

	//id, _ := res.LastInsertId()
	return nil
}

func OhlcUpdate(id int64, insertTime time.Time, symbol string, exchangeName string,
	kline Kline, updateTimes int) error {

	query := `REPLACE INTO ohlc5min (
		id, Symbol, OpenTime, Open, High, Low, Close, Volume, CloseTime,
		QuoteAssetVolume, NumberOfTrades, TakerBuyBaseAssetVolume, TakerBuyQuoteAssetVolume,
		exchangeName, insertTime, updateTime, UpdateTimes
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NOW(),?)`
	_, err := DBCon.Exec(query,
		id,
		symbol,
		kline.OpenTime.Format("2006-01-02 15:04:05"),
		kline.Open,
		kline.High,
		kline.Low,
		kline.Close,
		kline.Volume,
		kline.CloseTime.Format("2006-01-02 15:04:05"),
		kline.QuoteAssetVolume,
		kline.NumberOfTrades,
		kline.TakerBuyBaseAssetVolume,
		kline.TakerBuyQuoteAssetVolume,
		exchangeName,
		insertTime,
		updateTimes+1,
	)
	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return err
	}

	//idRet, _ := res.LastInsertId()
	//level.Debug(logger).Log("DBCon.Exec-idRet", idRet)

	return nil
}