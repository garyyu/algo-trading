package main

import (
	"os/exec"
	"github.com/go-kit/kit/log/level"
	"os"
	"fmt"
)

func RoiReport() {

	shellExec("",120.0)
	shellExec("",24.0)
	shellExec("",10.0)
	shellExec("",6.0)
	shellExec("",3.0)
	shellExec("",1.0)
	shellExec("",0.5)

	//cmd := "select u.Symbol,sum(u.Rank)/count(u.Rank) as AverageRank,count(u.Rank) as Count," +
	//	"max(u.Klines) as Klines,min(u.OpenTime) as OpenTime,max(u.EndTime) as EndTime," +
	//	"max(u.QuoteVol) as QuoteVol,max(u.Txs) as Txs from (select id,Symbol,Rank," +
	//	"round(InvestPeriod,1) as InvestPeriod,Klines,RoiD,RoiS,OpenTime,EndTime," +
	//	"QuoteAssetVolume as QuoteVol,NumberOfTrades as Txs from roi_5m " +
	//	"where Rank>0 order by id desc limit 21) as u group by u.Symbol order by AverageRank;"
	//
	//shellExec(cmd, 0.0)
}

func shellExec(cmdOverwrite string, investPeriod float32){

	var cmd string
	if len(cmdOverwrite)>0 {
		cmd = cmdOverwrite
	}else {
		cmd = "select id,Symbol,Rank,round(InvestPeriod,1) as InHours," +
			"RoiD,RoiS,OpenTime,EndTime,QuoteAssetVolume as QuoteVol," +
			"NumberOfTrades as Txs,Klines from roi_5m where Rank>0 and InvestPeriod=" +
			fmt.Sprintf("%.6f", investPeriod) +
			" order by id desc limit 3;"
	}

	mysqlLogin := "mysql -u" + os.Getenv("BINANCE_DB_USER") + " -p" +
		os.Getenv("BINANCE_DB_PWD") + " -Dbinance -t -e "

	stdout, err := exec.Command("sh","-c", mysqlLogin + "\"" + cmd + "\"").Output()
	if err != nil {
		level.Error(logger).Log("RoiReport.mysql", cmd, "err", err)
		return
	}
	fmt.Println("mysql> ", cmd, "\n\n", string(stdout)[1:])
}
