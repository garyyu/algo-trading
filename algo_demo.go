package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
)

type AlgoDemoConf struct {
	id				 		 int64		 `json:"id"`
	Symbol				 	 string		 `json:"Symbol"`
	Hours					 int 		 `json:"Hours"`
	StartTime            	 time.Time	 `json:"StartTime"`
}


func getAlgoDemoConf() map[string]AlgoDemoConf{

	algoDemoList := make(map[string]AlgoDemoConf)

	rows, err := DBCon.Query("select * from algo_demo")

	if err != nil {
		level.Error(logger).Log("getAlgoDemoConf - DBCon.Query fail! Err:", err)
		panic(err.Error())
	}
	defer rows.Close()

	algoDemoConf := AlgoDemoConf{}
	for rows.Next() {
		err := rows.Scan(&algoDemoConf.id, &algoDemoConf.Symbol, &algoDemoConf.Hours, &algoDemoConf.StartTime)
		if err != nil {
			level.Error(logger).Log("getAlgoDemoConf - rows.Scan Err:", err)
		}

		algoDemoList[algoDemoConf.Symbol] = algoDemoConf
	}

	return algoDemoList
}


