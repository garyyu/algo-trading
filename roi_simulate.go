package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"fmt"
	"sort"
	"strings"
)

const MaxKlinesMapSize = 1440		// for 5minutes klines, that's 5 days.

var (
	SymbolKlinesMapList []map[int64]KlineRo
	InvestPeriodList = [...]int{
		12,		// 1 Hour
		48,		// 4 Hour
		96,		// 8 Hour
		288,	// 24 Hour
		1440,	// 5 Days
		2880,	// 10 Days
		5760,	// 20 Days
	}
)


type RoiData struct {
	Symbol            		 string 	`json:"Symbol"`
	RoiRank					 int		`json:"RoiRank"`
	CashRatio			 	 float64	`json:"CashRatio"`
	InvestPeriod			 float64	`json:"InvestPeriod"`
	Klines 					 int		`json:"Klines"`
	RoiD					 float64	`json:"RoiD"`
	RoiS					 float64	`json:"RoiS"`
	QuoteAssetVolume         float64	`json:"QuoteAssetVolume"`
	NumberOfTrades           int		`json:"NumberOfTrades"`
	OpenTime                 time.Time	`json:"OpenTime"`
	EndTime                	 time.Time	`json:"EndTime"`
}

/*
 *  Main Routine for ROI Simulation
 */
func RoiRoutine(){

	fmt.Printf("RoiAnTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

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

			RoiSimulate()

			//RoiReport()

			/*
			 * Strictly limited ONLY for test !
			 * 		TODO: Auto-Creation of Project is Tricky! Wait for stable algorithm to select...
			 */
			//ProjectNew()
			//QueryOrders()

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
		roiList[i] = make([]RoiData, len(LivelySymbolList))
	}

	algoDemoList := getAlgoDemoConf()

	// calculation of ROI
	for i, symbol := range LivelySymbolList {

		klinesMap := SymbolKlinesMapList[i]

		var nowOpenTime= time.Time{}
		var nowCloseTime= time.Time{}
		var nowClose = 0.0

		// find the latest OpenTime
		for _, v := range klinesMap {
			if v.OpenTime.After(nowOpenTime) {
				nowOpenTime = v.OpenTime
				nowCloseTime = v.CloseTime
				nowClose = v.Close
			}
		}
		//fmt.Printf("%s - latest OpenTime=%s, size of klinesMap=%d\n", symbol,
		//	nowOpenTime.Format("2006-01-02 15:04:05"), len(klinesMap))

		if nowClose == 0.0 || len(klinesMap) < 5 {
			level.Error(logger).Log("RoiSimulate.Symbol", symbol, "nowClose", nowClose, "klinesMap", len(klinesMap))
			continue
		}

		for j, N := range InvestPeriodList {

			demo := false
			algoDemoConf,ok := algoDemoList[symbol]
			if ok && N==algoDemoConf.Hours*12 {
				demo = true
			}

			roiData := CalcRoi(symbol,
								N,
								nowOpenTime,
								nowCloseTime,
								nowClose,
								klinesMap,
								demo)
			roiList[j][i] = roiData

			//fmt.Printf("%s - CalcRoi(): klines used=%d for period %d\n",
			//	symbol, roiData.Klines, N)
		}
	}

	// RoiRank the ROI, to get Top 3 winners and Top 3 losers
	for p := range InvestPeriodList {

		// reverse the sequence
		j := len(InvestPeriodList)-1-p

		// Top 3 winners
		sort.Slice(roiList[j], func(m, n int) bool {
			return roiList[j][m].RoiD > roiList[j][n].RoiD
		})

		for q := range LivelySymbolList {

			// reverse the sequence
			i := len(LivelySymbolList)-1-q

			roiList[j][i].RoiRank = i + 1
			if i < 10{
				//fmt.Printf("RoiTop3Winer - %v\n", roiList[j][i])

				// Insert to Database
				InsertRoi(&roiList[j][i])

				if InvestPeriodList[j] <= 72 {	// [0.5 ~ 6.0] Hours
					HuntList[roiList[j][i].Symbol] = true
				}
			}
		}

		// Top 3 losers
		for i := len(LivelySymbolList)-3; i < len(LivelySymbolList); i++ {
			if i<0 {
				continue
			}

			roiList[j][i].RoiRank = i - len(LivelySymbolList)
			//fmt.Printf("RoiTop3Loser - %v\n", roiList[j][i])

			// Insert to Database
			InsertRoi(&roiList[j][i])
		}
	}

	// Insert HuntList to Database
	// TODO: strategy is to be designed more practical!
	//InsertHuntList(HuntList)
}

func CalcRoi(
		symbol string,
		N int,
		nowOpenTime time.Time,
		nowCloseTime time.Time,
		nowClose float64,
		klinesMap map[int64]KlineRo,
		demo bool) RoiData{

	var InitialAmount = 1000000.0   // in asset
	var balanceBase = InitialAmount // in asset
	var balanceQuote = 0.0    		// in btc

	var InitialOpen = 0.0
	var InitialOpenTime= time.Time{}
	var QuoteAssetVolume = 0.0
	var NumberOfTrades = 0

	// main algorithm
	var klinesUsed= 0
	for n := 1; n <= N; n++ {

		t0 := nowOpenTime.Add(time.Duration(-(N - n)*5) * time.Minute)
		t := t0.Unix()
		kline, ok := klinesMap[t]
		if !ok {
			//fmt.Println(symbol,"N=",N,"n=",n,"kline missing time:",
			//	t0.Format("2006-01-02 15:04:05"), " unix=", t, " on", -(N - n)*5)
			continue
		}

		klinesUsed += 1
		if InitialOpen == 0.0 {
			InitialOpen = kline.Open
			//InitialBalance = InitialAmount*InitialOpen
		}

		if InitialOpenTime.IsZero() {
			InitialOpenTime = kline.OpenTime
		}

		// core function call: auto-trading algorithm
		buy,sell := algoExample(&balanceBase, &balanceQuote, &kline, InitialAmount, InitialOpen, demo)
		if demo {
			fmt.Printf("CalcRoi - Demo: %s %s Ratio=%.1f%%. KLineRatio=%.1f%%, Sell=%.1f%%, Buy=%.1f%%. CashRatio=%.1f%%\n",
				symbol,
				kline.CloseTime.Format("2006-01-02 15:04:05"),
				(kline.Close - InitialOpen)/InitialOpen*100.0,
				(kline.Close - kline.Open)/InitialOpen*100.0,
				sell/InitialAmount*100.0, buy/InitialAmount*100.0,
				balanceQuote/(balanceBase*nowClose+balanceQuote)*100.0)
		}

		QuoteAssetVolume += kline.QuoteAssetVolume
		NumberOfTrades += kline.NumberOfTrades
	}

	// save the result
	roiD := (balanceBase*nowClose+balanceQuote)/(InitialAmount*InitialOpen) - 1.0
	roiS := nowClose/InitialOpen - 1.0
	roiData := RoiData{
		Symbol:       symbol,
		RoiRank:         0,
		CashRatio:	  balanceQuote/(balanceBase*nowClose+balanceQuote),
		InvestPeriod: float64(N) * 5.0 / 60.0,
		Klines:       klinesUsed,
		RoiD:         roiD,
		RoiS:		  roiS,
		QuoteAssetVolume: 	QuoteAssetVolume,
		NumberOfTrades: 	NumberOfTrades,
		OpenTime:			InitialOpenTime,
		EndTime:      		nowCloseTime,
	}

	if demo {
		fmt.Printf("\nCalcRoi - Demo Done. %s InHours=%.1f, RoiD=%.1f%%, RoiS=%.1f%%. CashRatio=%.1f%%.\n",
			symbol,
			roiData.InvestPeriod,
			roiD*100.0, roiS*100.0,
			roiData.CashRatio*100.0)
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

	query := `INSERT INTO roi_5m (
				Symbol, RoiRank, CashRatio, InvestPeriod, Klines, RoiD, RoiS, QuoteAssetVolume, NumberOfTrades, 
				OpenTime, EndTime, AnalysisTime
			  ) VALUES (?,?,?,?,?,?,?,?,?,?,?,NOW())`

	_, err := DBCon.Exec(query,
					roiData.Symbol,
					roiData.RoiRank,
					roiData.CashRatio,
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


