package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"fmt"
	"math"
	"sort"
	"strings"
)

const MaxKlinesMapSize int = 1440	// Minutes

var (
	SymbolKlinesMapList []map[int64]KlineRo
	InvestPeriodList = [...]int{6,12,36,72,120,288,1440}	// n*5 Mins
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
	RoiD					 float32	`json:"RoiD"`
	RoiS					 float32	`json:"RoiS"`
	QuoteAssetVolume         float64	`json:"QuoteAssetVolume"`
	NumberOfTrades           int		`json:"NumberOfTrades"`
	OpenTime                 time.Time	`json:"OpenTime"`
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

	fmt.Printf("\nRoiAnTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	// start a goroutine to get realtime ROI analysis in 1 min interval
	ticker := roiMinuteTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			ticker.Stop()

			tickerCount += 1
			fmt.Printf("RoiAnTick: \t\t%s\t%d\n", tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)
			//hour, min, sec := tick.Clock()

			PollKlines()

			RoiSimulate()

			RoiReport()

			/*
			 * Strictly limited ONLY for test !
			 */
			ProjectNew()

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
func RoiSimulate() {

	HuntList :=  make(map[string]bool)

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
		//fmt.Println("latest OpenTime=", nowOpenTime)

		if nowClose == 0.0 || len(klinesMap) < 5 {
			level.Error(logger).Log("RoiSimulate.Symbol", symbol, "nowClose", nowClose, "klinesMap", len(klinesMap))
			continue
		}

		for j, N := range InvestPeriodList {

			var Initial = 10000.0     // in asset
			var balanceBase = Initial // in asset
			var balanceQuote = 0.0    // in btc

			var InitialOpen = 0.0
			var InitialOpenTime= time.Time{}
			var QuoteAssetVolume = 0.0
			var NumberOfTrades = 0

			// main algorithm
			var klinesUsed= 0
			for n := 1; n <= N; n++ {

				t := nowOpenTime.Add(time.Duration(-(N - n)*5) * time.Minute).Unix()
				kline, ok := klinesMap[t]
				if !ok {
					//fmt.Println(symbol,"N=",N,"n=",n,"kline missing @ time:", t, " on", -(N - n)*5)
					continue
				}

				klinesUsed += 1
				if InitialOpen == 0.0 {
					InitialOpen = kline.Open
				}

				if InitialOpenTime.IsZero() {
					InitialOpenTime = kline.OpenTime
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

				QuoteAssetVolume += kline.QuoteAssetVolume
				NumberOfTrades += kline.NumberOfTrades
			}

			// save the result
			roiD := (balanceBase*nowClose+balanceQuote)/(Initial*InitialOpen) - 1.0
			roiS := nowClose/InitialOpen - 1.0
			roiData := RoiData{
				Symbol:       symbol,
				Rank:         0,
				InvestPeriod: float32(N) * 5.0 / 60.0,
				Klines:       klinesUsed,
				RoiD:         float32(roiD),
				RoiS:		  float32(roiS),
				QuoteAssetVolume: 	QuoteAssetVolume,
				NumberOfTrades: 	NumberOfTrades,
				OpenTime:			InitialOpenTime,
				EndTime:      		nowCloseTime,
			}
			roiList[j][i] = roiData
		}
	}

	// Rank the ROI, to get Top 3 winners and Top 3 losers
	for p := range InvestPeriodList {

		// reverse the sequence
		j := len(InvestPeriodList)-1-p

		// Top 3 winners
		sort.Slice(roiList[j], func(m, n int) bool {
			return roiList[j][m].RoiD > roiList[j][n].RoiD
		})

		for q := range SymbolList {

			// reverse the sequence
			i := len(SymbolList)-1-q

			roiList[j][i].Rank = i + 1
			if i < 3 {
				//fmt.Printf("RoiTop3Winer - %v\n", roiList[j][i])

				// Insert to Database
				InsertRoi(&roiList[j][i])

				if InvestPeriodList[j] <= 72 {	// [0.5 ~ 6.0] Hours
					HuntList[roiList[j][i].Symbol] = true
				}
			}
		}

		// Top 3 losers
		for i := len(SymbolList)-3; i < len(SymbolList); i++ {
			if i<0 {
				continue
			}

			roiList[j][i].Rank = i - len(SymbolList)
			//fmt.Printf("RoiTop3Loser - %v\n", roiList[j][i])

			// Insert to Database
			InsertRoi(&roiList[j][i])
		}
	}

	// Insert HuntList to Database
	InsertHuntList(HuntList)
}

/*
 * Insert ROI result into Database
 */
func InsertRoi(roiData *RoiData){

	if roiData==nil || roiData.EndTime.IsZero() {
		level.Warn(logger).Log("InsertRoi.roiData", roiData)
		return
	}

	query := `INSERT INTO roi5min (
				Symbol, Rank, InvestPeriod, Klines, RoiD, RoiS, QuoteAssetVolume, NumberOfTrades, 
				OpenTime, EndTime, AnalysisTime
			  ) VALUES (?,?,?,?,?,?,?,?,?,?,NOW())`

	_, err := DBCon.Exec(query,
					roiData.Symbol,
					roiData.Rank,
					roiData.InvestPeriod,
					roiData.Klines,
					roiData.RoiD,
					roiData.RoiS,
					roiData.QuoteAssetVolume,
					roiData.NumberOfTrades,
					roiData.OpenTime,
					roiData.EndTime,
				)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return
	}

	//id, _ := res.LastInsertId()
}

/*
 * Insert HuntList into Database
 */
func InsertHuntList(huntList map[string]bool){

	sqlStr := "INSERT INTO hunt_list (Symbol, Time) VALUES "
	var vals []interface{}

	for symbol, hunt := range huntList {
		if hunt {
			sqlStr += "(?, NOW()),"
			vals = append(vals, symbol)
		}
	}
	//trim the last ,
	sqlStr = strings.TrimSuffix(sqlStr, ",")

	stmt, err := DBCon.Prepare(sqlStr)
	if err != nil {
		level.Error(logger).Log("DBCon.Prepare", err)
		return
	}
	_, err2 := stmt.Exec(vals...)
	if err2 != nil {
		level.Error(logger).Log("DBCon.Exec", err2)
		return
	}
}


