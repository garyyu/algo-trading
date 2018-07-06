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

			QueryOrders()

			//TODO: should be in an self thread
			ProjectManager()

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

			roiData := CalcRoi(symbol,
								N,
								nowOpenTime,
								nowCloseTime,
								nowClose,
								klinesMap)
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
			if i < 3{
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

func CalcRoi(
		symbol string,
		N int,
		nowOpenTime time.Time,
		nowCloseTime time.Time,
		nowClose float64,
		klinesMap map[int64]KlineRo) RoiData{

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
			if sell < MinOrderTotal { // note: $8 = 0.001btc on $8k/btc
				sell = 0
			}
		} else if gain < 0 {
			buy = math.Min(balanceQuote, -gain*kline.Close)
			if buy < MinOrderTotal {  // note: $8 = 0.001btc on $8k/btc
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

	return roiData
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


