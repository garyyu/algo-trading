package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"bitbucket.org/garyyu/go-binance"
	"fmt"
	"os"
	"os/exec"
)

type CsvPollType int

const (
	Minute         CsvPollType = iota
	ThreeMinutes
	FiveMinutes
	FifteenMinutes
	ThirtyMinutes
	Hour
	TwoHours
	FourHours
	SixHours
	EightHours
	TwelveHours
	Day
	ThreeDays
	Week
	Month
)

type CsvPollConf struct {
	id				 		 int64		 `json:"id"`
	Symbol				 	 string		 `json:"Symbol"`
	Type					 CsvPollType `json:"Type"`
	StartTime            	 time.Time	 `json:"StartTime"`
}

func csvPolling(symbol string, interval binance.Interval) bool{

	const MaxLinesAllowed = 1000

	OpenTime := GetOpenTime(symbol, interval)
	if OpenTime.IsZero() {
		fmt.Printf("csvPolling - Fail. OpenTime is Zero. Symbol=%s, interval=%s",
			symbol, interval)
		return false
	}

	switch interval{
	case binance.FiveMinutes:
		OpenTime = OpenTime.Add(-5*time.Minute*(MaxLinesAllowed-1))
	case binance.Hour:
		OpenTime = OpenTime.Add(-60*time.Minute*(MaxLinesAllowed-1))
	case binance.Day:
		OpenTime = OpenTime.Add(-24*60*time.Minute*(MaxLinesAllowed-1))
	default:
		fmt.Printf("csvPolling - Fail. Interval not supported. Symbol=%s, interval=%s",
			symbol, interval)
		return false
	}

	// remove existing
	filename := "~/tmp/" + symbol + "-" + string(interval) + ".csv"
	_, err := exec.Command("sh","-c", "rm -f " + filename).Output()
	if err != nil {
		level.Error(logger).Log("csvPolling - removing existing output fail! file:",
			filename, "err:", err)
		return false
	}

	cmd := "select OpenTime,Open,High,Low,Close,Volume,QuoteAssetVolume from ohlc_" + string(interval) +
		" where Symbol='" + symbol + "' and OpenTime>='" + OpenTime.Format("2006-01-02 15:04:05") +
		"' order by OpenTime INTO OUTFILE '" + filename + "'"

	mysqlLogin := "mysql -u" + os.Getenv("BINANCE_DB_USER") + " -p" +
		os.Getenv("BINANCE_DB_PWD") + " -Dbinance -e "

	_, err2 := exec.Command("sh","-c", mysqlLogin + "\"" + cmd + "\"").Output()
	if err2 != nil {
		level.Error(logger).Log("csvPolling.mysql", cmd, "err", err2)
		return false
	}

	return true
}


func getCsvPollConf(interval binance.Interval) map[string]CsvPollConf{
	//
	//fmt.Printf("getCsvPollConf - func enter. interval=%s\n", string(interval))

	csvPollList := make(map[string]CsvPollConf)

	var intervalStr string
	switch interval {
	case binance.FiveMinutes:
		intervalStr = "FiveMinutes"
	case binance.Hour:
		intervalStr = "Hour"
	case binance.Day:
		intervalStr = "Day"
	default:
		fmt.Printf("getCsvPollConf - Fail. interval not supported: %s\n", string(interval))
		return csvPollList
	}

	rows, err := DBCon.Query("select * from csv_polling where Type=?", intervalStr)

	if err != nil {
		level.Error(logger).Log("getCsvPollConf - DBCon.Query fail! Err:", err)
		panic(err.Error())
	}
	defer rows.Close()

	csvPollConf := CsvPollConf{}
	var pollType string
	for rows.Next() {
		err := rows.Scan(&csvPollConf.id, &csvPollConf.Symbol, &pollType, &csvPollConf.StartTime)
		if err != nil {
			level.Error(logger).Log("getCsvPollConf - rows.Scan Err:", err)
		}

		switch pollType {
		case "FiveMinutes":	csvPollConf.Type = FiveMinutes
		case "Hour": csvPollConf.Type = Hour
		case "Day": csvPollConf.Type = Day
		default: 	csvPollConf.Type = FiveMinutes
		}


		csvPollList[csvPollConf.Symbol] = csvPollConf
	}
	//
	//fmt.Printf("getCsvPollConf - func exit. csvPollList=%v\n", csvPollList)

	return csvPollList
}


