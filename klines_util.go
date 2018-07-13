package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"fmt"
	"database/sql"
	"bitbucket.org/garyyu/algo-trading/go-binance"
)

func getKlineId(symbol string, openTime time.Time, table string) (int64,time.Time,int){

	rows, err := DBCon.Query("SELECT id,insertTime,UpdateTimes FROM " + table +
		" WHERE Symbol='" + symbol + "' and OpenTime='" +
		openTime.Format("2006-01-02 15:04:05") + "' limit 1")

	//level.Debug(logger).Log("getKlinesId.Query", openTime.Format("2006-01-02 15:04:05"))

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	var id int64 = -1	// if not found, rows is empty.
	var insertTime time.Time
	var updateTimes = 0
	for rows.Next() {
		err := rows.Scan(&id, &insertTime, &updateTimes)
		if err != nil {
			level.Error(logger).Log("getKlineId.err", err)
			id = -1
		}
	}
	//fmt.Println("getKlinesId() for", symbol, "at time",
	//	openTime.Format("2006-01-02 15:04:05"), " id=", id,
	//	"insertTime=", insertTime.Format("2006-01-02 15:04:05"))
	return id,insertTime,updateTimes
}

func getKlinesData(symbol string, limit int, interval binance.Interval) (int,int){

	var rowsNum = 0
	var rowsNewNum = 0
	var retry = 0
	for {
		retry += 1

		kl, err := binanceSrv.Klines(binance.KlinesRequest{
			Symbol:   symbol,
			Interval: interval,
			Limit:    limit,
		})
		if err != nil {
			level.Error(logger).Log("getKlinesData.Symbol", symbol, "Err", err, "Retry", retry-1)
			if retry >= 10 {
				break
			}

			switch retry {
			case 1:
				time.Sleep(1 * time.Second)
			case 2:
				time.Sleep(3 * time.Second)
			case 3:
				time.Sleep(5 * time.Second)
			case 4:
				time.Sleep(10 * time.Second)
			default:
				time.Sleep(15 * time.Second)
			}
			continue
		}

		if limit > 3 {
			fmt.Printf("%s - %s received %d %s-klines\n",
				time.Now().Format("2006-01-02 15:04:05.004005683"), symbol, len(kl), string(interval))
		}
		for _, v := range kl {
			rowsNum += 1
			id,insertTime,updateTimes := getKlineId(symbol, v.OpenTime, "ohlc_" + string(interval))
			if id < 0 {
				OhlcCreate(symbol, "binance.com", *v,
					"ohlc_" + string(interval))
				rowsNewNum += 1
			} else {
				// update it
				OhlcUpdate(id, insertTime, symbol, "binance.com", *v, updateTimes,
					"ohlc_" + string(interval))
			}
		}

		break
	}

	return rowsNum,rowsNewNum
}

/*
 * Get Latest one kline from local database
 */
func GetTimePrice(symbol string) TimePrice {

	row := DBCon.QueryRow(
		"SELECT CloseTime,Close FROM ohlc_5m WHERE Symbol=? order by id desc limit 1",
		symbol)

	timePrice := TimePrice{}
	err := row.Scan(&timePrice.TradeTime, &timePrice.Price)
	if err != nil && err != sql.ErrNoRows {
		level.Error(logger).Log("GetTimePrice - Scan Err:", err)
	}

	// the latest kline data 'CloseTime' always as 'OpenTime'+5 Minutes, use Now() time instead.
	if timePrice.TradeTime.After(time.Now()) {
		timePrice.TradeTime = time.Now()
	}

	return timePrice
}


/*
 * Get Latest kline OpenTime from local database
 */
func GetOpenTime(symbol string, interval binance.Interval) time.Time {

	query := "SELECT OpenTime FROM ohlc_" + string(interval) +
		" WHERE Symbol=? order by OpenTime desc limit 1"

	row := DBCon.QueryRow(query, symbol)

	openTime := time.Time{}
	err := row.Scan(&openTime)
	if err != nil && err != sql.ErrNoRows {
		level.Error(logger).Log("GetOpenTime - row.Scan Err:", err)
	}

	return openTime
}

