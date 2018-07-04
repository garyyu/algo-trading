package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"fmt"
	"math"
	"sort"
)

const MaxKlinesMapSize int = 1440	// Minutes

var (
	SymbolKlinesMapList []map[int64]KlineRo
	InvestPeriodList = [...]int{6,12,36,72,120,600,1440}	// n*5 Mins
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

type RoiData struct {
	Symbol            		 string 	`json:"Symbol"`
	Rank					 int		`json:"Rank"`
	InvestPeriod			 float32	`json:"InvestPeriod"`
	Klines 					 int		`json:"Klines"`
	Roi						 float32	`json:"Roi"`
	EndTime                	 time.Time	`json:"EndTime"`
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

/*
 *  Main Routine for ROI Simulation
 */
func RoiRoutine(){

	InitKlines()

	fmt.Printf("\nRoiAnaTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	// start a goroutine to get realtime ROI analysis in 1 min interval
	ticker := roiMinuteTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			tickerCount += 1
			fmt.Printf("RoiAnaTick: \t\t%s\t%d\n", tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)
			//hour, min, sec := tick.Clock()

			PollKlines()

			RoiSimulate(tickerCount)

			// Update the ticker
			ticker = minuteTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - RoiRoutine")
}

func roiMinuteTicker() *time.Ticker {
	// Return new ticker that triggers on the minute
	now := time.Now()
	second := 30 - now.Second()		// range: [30..-29]
	if second <= 0 {
		second += 60				// range: [30..0] + [60..31] -> [0..60]
	}

	return time.NewTicker(
		time.Second * time.Duration(second) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}


/*
 * ROI (Return Of Invest) Simulation
 */
func RoiSimulate(tickerCount int) {

	// allocate result 2D array
	var roiList= make([][]RoiData, len(InvestPeriodList))
	for i := range roiList {
		roiList[i] = make([]RoiData, len(SymbolList))
	}

	// calculation of ROI
	for i, symbol := range SymbolList {

		klinesMap := SymbolKlinesMapList[i]

		var nowOpenTime= time.Time{}
		var nowCloseTime= time.Time{}
		var nowClose float64 = 0.0

		// find the latest OpenTime
		for _, v := range klinesMap {
			if v.OpenTime.After(nowOpenTime) {
				nowOpenTime = v.OpenTime
				nowCloseTime = v.CloseTime
				nowClose = v.Close
			}
		}

		if nowClose == 0.0 || len(klinesMap) < 5 {
			level.Error(logger).Log("RoiSimulate.Symbol", symbol, "nowClose", nowClose, "klinesMap", len(klinesMap))
		}

		for j, N := range InvestPeriodList {

			var Initial float64 = 10000.0     // in asset
			var balanceBase float64 = Initial // in asset
			var balanceQuote float64 = 0.0    // in btc

			var InitialOpen float64 = 0.0

			// main algorithm
			var klinesUsed= 0
			for n := 1; n <= N; n++ {

				t := nowOpenTime.Add(time.Duration(-(N - n)*5) * time.Minute).Unix()
				kline, ok := klinesMap[t]
				if !ok {
					continue
				}

				klinesUsed += 1
				if InitialOpen == 0.0 {
					InitialOpen = kline.Open
				}

				sell := 0.0
				buy := 0.0
				gain := (kline.Close - kline.Open) / kline.Open * balanceBase
				if gain > 0 {
					sell = gain * kline.Close
					if sell < 0.00002 { // note: $0.1 = 0.0000125btc on $8k/btc
						sell = 0.0
					}
				} else if gain < 0 {
					buy = math.Min(balanceQuote, -gain*kline.Close)
					if buy < 0.00002 { // note: $0.1 = 0.0000125btc on $8k/btc
						buy = 0
					}
				}

				balanceQuote += sell - buy
				balanceBase += gain - (sell-buy)/kline.Close
			}

			// save the result
			roi := (balanceBase*nowClose+balanceQuote)/(Initial*InitialOpen) - 1.0
			roiData := RoiData{
				Symbol:       symbol,
				Rank:         0,
				InvestPeriod: float32(N) * 5.0 / 60.0,
				Klines:       klinesUsed,
				Roi:          float32(roi),
				EndTime:      nowCloseTime,
			}
			roiList[j][i] = roiData
		}
	}

	// Rank the ROI, to get Top 3 winners and Top 3 losers
	for j := range InvestPeriodList {

		// Top 3 winners
		sort.Slice(roiList[j], func(m, n int) bool {
			return roiList[j][m].Roi > roiList[j][n].Roi
		})

		for i := range SymbolList {
			roiList[j][i].Rank = i + 1
			if i < 3 {
				//fmt.Printf("RoiTop3Winer - %v\n", roiList[j][i])

				// Insert to Database
				InsertRoi(&roiList[j][i], tickerCount)
			}
		}

		// Top 3 losers
		for i := len(SymbolList)-1; i >= len(SymbolList)-3; i-- {
			if i<0 {
				continue
			}

			roiList[j][i].Rank = i - len(SymbolList)
			//fmt.Printf("RoiTop3Loser - %v\n", roiList[j][i])

			// Insert to Database
			InsertRoi(&roiList[j][i], tickerCount)
		}
	}
}

/*
 * Insert ROI result into Database
 */
func InsertRoi(roiData *RoiData, tickerCount int){

	query := `INSERT INTO roi5min (
				Symbol, Rank, InvestPeriod, Klines, Roi, EndTime, TickerCount
			  ) VALUES (?,?,?,?,?,?,?)`

	_, err := DBCon.Exec(query,
					roiData.Symbol,
					roiData.Rank,
					roiData.InvestPeriod,
					roiData.Klines,
					roiData.Roi,
					roiData.EndTime,
					tickerCount,
				)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return
	}

	//id, _ := res.LastInsertId()
}


