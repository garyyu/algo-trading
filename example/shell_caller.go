package main

import (
	"os/exec"
	"github.com/go-kit/kit/log/level"
	"os"
	"fmt"
)

func RoiReport() {

	shellExec(120.0)
	shellExec(50.0)
	shellExec(10.0)
	shellExec(6.0)
	shellExec(3.0)
	shellExec(1.0)
	shellExec(0.5)
}

func shellExec(investPeriod float32){

	cmd := "select id,Symbol,Rank,round(InvestPeriod,1) as Period(Hour)," +
		"RoiD,RoiS,OpenTime,EndTime,QuoteAssetVolume as QuoteVol," +
		"NumberOfTrades as Txs,Klines from roi5min where Rank>0 and InvestPeriod=" +
		fmt.Sprintf("%.6f", investPeriod) +
		" order by id desc limit 3;"

	mysqlLogin := "mysql -u" + os.Getenv("BINANCE_DB_USER") + " -p" +
		os.Getenv("BINANCE_DB_PWD") + " -Dbinance -t -e "

	stdout, err := exec.Command("sh","-c", mysqlLogin + "\"" + cmd + "\"").Output()
	if err != nil {
		level.Error(logger).Log("RoiReport.cmd", cmd, "err", err)
		return
	}
	fmt.Println("mysql> ", cmd, "\n\n", string(stdout)[1:])
}
