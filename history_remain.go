package main

import (
	"github.com/go-kit/kit/log/level"
	"time"
	"database/sql"
)

type HistoryRemain struct {
	id				 int64		`json:"id"`
	Symbol           string 	`json:"Symbol"`
	Amount			 float64	`json:"Amount"`
	Time             time.Time	`json:"Time"`
}

/*
 * Get Account History Remain from local database
 */
func GetHistoryRemain(symbol string) HistoryRemain{

	query := "select * from history_remain where Symbol='" + symbol + "'"

	row := DBCon.QueryRow(query)

	historyRemain := HistoryRemain{}

	err := row.Scan(&historyRemain.id, &historyRemain.Symbol,
		&historyRemain.Amount, &historyRemain.Time)

	if err != nil && err != sql.ErrNoRows  {
		level.Error(logger).Log("getHistoryRemain - Scan Err:", err)
	}

	return historyRemain
}


/*
 * Update History Remaining Amount into Database
 */
func UpdateHistoryRemain(symbol string, amount float64) bool{

	query := `INSERT INTO history_remain (Symbol, Amount, Time) VALUES (?,?,NOW())
			  ON DUPLICATE KEY UPDATE Symbol=?, Amount=?, Time=NOW()`

	res, err := DBCon.Exec(query,
		symbol,
		amount,
		symbol,
		amount,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return false
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected>=0 {
		return true
	}else{
		return false
	}
}
