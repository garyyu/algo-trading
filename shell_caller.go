package main

import (
	"os/exec"
	"github.com/go-kit/kit/log/level"
	"os"
	"fmt"
)

func RoiReport() {

	//shellExec("",480.0)
	//shellExec("",240.0)
	shellExec("",120.0)
	shellExec("",24.0)
	shellExec("",8.0)
	shellExec("",4.0)
	shellExec("",1.0)

	//cmd := "select u.Symbol,sum(u.RoiRank)/count(u.RoiRank) as AverageRoiRank,count(u.RoiRank) as Count," +
	//	"max(u.Klines) as Klines,min(u.OpenTime) as OpenTime,max(u.EndTime) as EndTime," +
	//	"max(u.QuoteVol) as QuoteVol,max(u.Txs) as Txs from (select id,Symbol,RoiRank," +
	//	"round(InvestPeriod,1) as InvestPeriod,Klines,RoiD,RoiS,OpenTime,EndTime," +
	//	"QuoteAssetVolume as QuoteVol,NumberOfTrades as Txs from roi_5m " +
	//	"where RoiRank>0 order by id desc limit 21) as u group by u.Symbol order by AverageRoiRank;"
	//
	//shellExec(cmd, 0.0)

}

func HotspotReport(){

	cmd := "select Symbol,HotRank,concat(Round(HighLowRatio*100,1),'%') as HighLowRatio,"+
		"concat(Round(VolumeRatio*100,1),'%') as VolumeRatio,"+
		"concat(Round(CloseOpenRatio*100,1),'%') as CloseOpenRatio,HLRxVR,Time,"+
		"if(HLRxVR>2,'Fever',if(HLRxVR>0.2,'Hot','')) as Level "+
		"from hotspot_5m order by id desc limit 18;"
	shellExec(cmd, 0.0)
}

func shellExec(cmdOverwrite string, investPeriod float32){

	var cmd string
	if len(cmdOverwrite)>0 {
		cmd = cmdOverwrite
	}else {
		cmd = "select id,Symbol,RoiRank,concat(Round(CashRatio * 100,1), '%') as CashRatio,"+
			"round(InvestPeriod,1) as InHours," +
			"RoiD,RoiS,OpenTime,EndTime,QuoteAssetVolume as QuoteVol," +
			"NumberOfTrades as Txs,Klines from roi_5m where RoiRank>0 and InvestPeriod=" +
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
	fmt.Println("mysql> ", cmd, "\n\n", string(stdout))
}
