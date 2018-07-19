package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"fmt"
	"sort"
	"math"
	"sync"
)

type HotspotData struct {
	Symbol            		 string 	`json:"Symbol"`
	HotRank					 int		`json:"HotRank"`
	HighLowRatio			 float64	`json:"HighLowRatio"`
	VolumeRatio			 	 float64	`json:"VolumeRatio"`
	CloseOpenRatio			 float64	`json:"CloseOpenRatio"`
	HLRxVR			 		 float64	`json:"HLRxVR"`
	Time                	 time.Time	`json:"Time"`
}


/*
 *  Main Routine for Hotlist Hunting
 */
func HotspotRoutine(wg *sync.WaitGroup){
	defer wg.Done()

	fmt.Printf("HotspotTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	ticker := hotspotMinuteTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop
		case tick := <-ticker.C:
			ticker.Stop()

			if tickerCount % 5 == 0 {	// temporary change to slower ticker: 5 minutes

				fmt.Printf("HotlistTick: \t\t%s\t%d\n", tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)

				HotspotSearch()
				HotspotReport()
			}

			tickerCount += 1

			// Update the ticker
			ticker = hotspotMinuteTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - HotspotRoutine")
}

func hotspotMinuteTicker() *time.Ticker {

	now := time.Now()
	second := 10 - now.Second()		// range: [10..-49]
	if second <= 0 {
		second += 60				// range: [10..0] + [60..11] -> [0..60]
	}

	return time.NewTicker(
		time.Second * time.Duration(second) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}


/*
 * Hotspot Searching
 */
func HotspotSearch() {

	hotspotList := make([]HotspotData, len(LivelySymbolList))

	Klines5mMutex.RLock()

	for i, symbol := range LivelySymbolList {

		klinesMap := SymbolKlinesMapList[i]

		var nowOpenTime = time.Time{}
		maxVolume := 0.0

		// find the latest OpenTime and max volume
		for _, v := range klinesMap {
			if v.OpenTime.After(nowOpenTime) {
				nowOpenTime = v.OpenTime
			}

			maxVolume = math.Max(v.Volume, maxVolume)
		}

		if nowOpenTime.IsZero() {
			return
		}

		// previous kline OpenTime
		prevOpenTime := nowOpenTime.Add(time.Minute * -5).Unix()
		if _, ok := klinesMap[prevOpenTime]; !ok{
			return
		}

		currOpenTime := nowOpenTime.Unix()

		// calculation

		HighLowRatio := klinesMap[currOpenTime].High / klinesMap[currOpenTime].Low - 1.0
		VolumeRatio := klinesMap[currOpenTime].Volume / maxVolume

		diff := klinesMap[currOpenTime].Close - klinesMap[prevOpenTime].Close
		if diff < 0{
			HighLowRatio = -HighLowRatio
			VolumeRatio = -VolumeRatio
		}

		CloseOpenRatio := klinesMap[currOpenTime].Close / klinesMap[currOpenTime].Open - 1.0
		HLRxVR := HighLowRatio * 100.0 * VolumeRatio

		// save the result
		hotspotData := HotspotData{
			Symbol:       	symbol,
			HighLowRatio: 	HighLowRatio,
			VolumeRatio:	VolumeRatio,
			CloseOpenRatio:	CloseOpenRatio,
			HLRxVR:			HLRxVR,
		}

		hotspotList = append(hotspotList, hotspotData)
	}

	Klines5mMutex.RUnlock()

	// Sort them on VolumeRatio
	sort.Slice(hotspotList, func(m, n int) bool {
		return math.Abs(hotspotList[m].VolumeRatio) > math.Abs(hotspotList[n].VolumeRatio)
	})

	// Saving Top 3 winners on VolumeRatio
	for q := range LivelySymbolList {

		// reverse the sequence
		i := len(LivelySymbolList)-1-q

		hotspotList[i].HotRank = i + 1 + 100000
		if i < 3 {
			// Insert to Database
			InsertHotspot(&hotspotList[i])
		}
	}

	// Sort them on HighLowRatio
	sort.Slice(hotspotList, func(m, n int) bool {
		return math.Abs(hotspotList[m].HighLowRatio) > math.Abs(hotspotList[n].HighLowRatio)
	})

	// Saving Top 3 winners on HighLowRatio
	for q := range LivelySymbolList {

		// reverse the sequence
		i := len(LivelySymbolList)-1-q

		hotspotList[i].HotRank = i + 1 + 1000
		if i < 3 {
			// Insert to Database
			InsertHotspot(&hotspotList[i])
		}
	}

	// Sort them on HLRxVR
	sort.Slice(hotspotList, func(m, n int) bool {
		return hotspotList[m].HLRxVR > hotspotList[n].HLRxVR
	})

	// Saving Top 3 winners on HLRxVR
	for q := range LivelySymbolList {

		// reverse the sequence
		i := len(LivelySymbolList)-1-q

		hotspotList[i].HotRank = i + 1
		if i < 3 {
			// Insert to Database
			InsertHotspot(&hotspotList[i])
		}
	}

}

/*
 * Insert Hotspot result into Database
 */
func InsertHotspot(hotspotData *HotspotData) int64{

	query := `INSERT INTO hotspot_5m (
				Symbol, HotRank, HighLowRatio, VolumeRatio, CloseOpenRatio, HLRxVR, Time
			  ) VALUES (?,?,?,?,?,?,NOW())`

	res, err := DBCon.Exec(query,
		hotspotData.Symbol,
		hotspotData.HotRank,
		hotspotData.HighLowRatio,
		hotspotData.VolumeRatio,
		hotspotData.CloseOpenRatio,
		hotspotData.HLRxVR,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return -1
	}

	id, _ := res.LastInsertId()
	return id
}

