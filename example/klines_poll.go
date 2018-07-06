package main

import (
	"github.com/go-kit/kit/log/level"
	"time"
	"fmt"
)

type KlineRo struct {
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
}

func InitKlines() {

	fmt.Println("Initizing Klines from database ...\t\t\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

	sqlStatement := `SELECT id,Symbol,OpenTime,Open,High,Low,Close,Volume,CloseTime,
				QuoteAssetVolume,NumberOfTrades,TakerBuyBaseAssetVolume,TakerBuyQuoteAssetVolume
				FROM ohlc5min WHERE Symbol=? order by OpenTime desc limit 1440;`

	// Initialize the global 'SymbolKlinesMapList'
	totalSymbols := len(SymbolList)
	SymbolKlinesMapList = make([]map[int64]KlineRo, totalSymbols)
	for i:=0; i<totalSymbols; i++ {
		SymbolKlinesMapList[i] =  make(map[int64]KlineRo)
	}

	// Query database
	var totalQueryRet = 0
	for i,symbol := range SymbolList {

		rows, err := DBCon.Query(sqlStatement, symbol)

		if err != nil {
			level.Error(logger).Log("DBCon.Query", err)
			panic(err)
		}

		klinesMap := SymbolKlinesMapList[i]
		var rowsNum = 0
		for rows.Next() {
			rowsNum += 1
			var klineRo KlineRo

			err := rows.Scan(&klineRo.Id, &klineRo.Symbol, &klineRo.OpenTime,
				&klineRo.Open, &klineRo.High, &klineRo.Low, &klineRo.Close,
				&klineRo.Volume, &klineRo.CloseTime, &klineRo.QuoteAssetVolume, &klineRo.NumberOfTrades,
				&klineRo.TakerBuyBaseAssetVolume, &klineRo.TakerBuyQuoteAssetVolume)

			if err != nil {
				level.Error(logger).Log("rows.Scan", err)
				break
			}

			klinesMap[klineRo.OpenTime.Unix()] = klineRo
		}
		//fmt.Println("InitKlines - ", symbol, "got", rowsNum)
		totalQueryRet += rowsNum
	}
	fmt.Println("InitKlines - ", len(SymbolList), "symbols", " average Klines number:",
		float32(totalQueryRet)/float32(len(SymbolList)),
		"\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

}

/*
 * This is a periodic polling from database for latest Klines, in 1 minute interval.
 */
func PollKlines() {

	fmt.Println("Polling Klines from database ...\t\t\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))

	sqlStatement := `SELECT id,Symbol,OpenTime,Open,High,Low,Close,Volume,CloseTime,
				QuoteAssetVolume,NumberOfTrades,TakerBuyBaseAssetVolume,TakerBuyQuoteAssetVolume
				FROM ohlc5min WHERE Symbol=? order by OpenTime desc limit 2;`

	var totalQueryRet = 0
	for i,symbol := range SymbolList {

		rows, err := DBCon.Query(sqlStatement, symbol)

		if err != nil {
			level.Error(logger).Log("DBCon.Query", err)
			panic(err)
		}

		klinesMap := SymbolKlinesMapList[i]
		var rowsNum = 0
		for rows.Next() {
			rowsNum += 1
			var klineRo KlineRo

			err := rows.Scan(&klineRo.Id, &klineRo.Symbol, &klineRo.OpenTime,
				&klineRo.Open, &klineRo.High, &klineRo.Low, &klineRo.Close,
				&klineRo.Volume, &klineRo.CloseTime, &klineRo.QuoteAssetVolume, &klineRo.NumberOfTrades,
				&klineRo.TakerBuyBaseAssetVolume, &klineRo.TakerBuyQuoteAssetVolume)

			if err != nil {
				level.Error(logger).Log("rows.Scan", err)
				break
			}

			// note: it could overwrite an existing kline, if database query got duplicated ones.
			klinesMap[klineRo.OpenTime.Unix()] = klineRo

			// check if map is over 'MaxKlinesMapSize' limit and prune it
			oldestKlineTime := klineRo.OpenTime.Add(time.Duration(-MaxKlinesMapSize-5) * time.Minute).Unix()
			if _, ok := klinesMap[oldestKlineTime]; ok {
				delete(klinesMap, oldestKlineTime)
			}
		}
		//fmt.Println("PollKlines - ", symbol, "got", rowsNum)
		totalQueryRet += rowsNum
	}

	fmt.Println("PollKlines - ", len(SymbolList), "symbols", " average Klines number:",
		float32(totalQueryRet)/float32(len(SymbolList)),
		"\t\t", time.Now().Format("2006-01-02 15:04:05.004005683"))
}

