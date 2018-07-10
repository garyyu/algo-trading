package main

import (
	"fmt"
	"github.com/go-kit/kit/log/level"
	"database/sql"
	"time"
	"math"
)

/*
 * Robot Project Manager
 */
func ProjectManager(){

	// Project Performance Indicator Refreshing
	ProjectMutex.Lock()
	for _, project := range ActiveProjectList {

		if project.IsClosed {
			continue
		}

		// get latest price
		highestBid := OBData{}
		highestBid = getHighestBid(project.Symbol)
		if highestBid.Time.Add(time.Second * 60).Before(time.Now()) {
			fmt.Println("Warning! ProjectManager - getHighestBid got old data. can't manage project: ", project.Symbol)
			continue
		}

		// import all orders into local database, and map them into related projects
		GetAllOrders(project.Symbol)

		netBuy,netIncome := GetProjectNetBuy(project.id)
		if netBuy == 0 && netIncome == 0 {
			fmt.Printf("ProjectManager - %s order info not finalized! wait next ticker.\n",
				project.Symbol)
			continue
		}else{
			//fmt.Printf("ProjectManager - %s: NetBuy=%f, NetIncome=%f\n",
			//	project.Symbol, netBuy, netIncome)
		}

		if netBuy > 0 && !FloatEquals(project.BalanceBase,netBuy) {

			fmt.Printf("Warning! ProjectManager - %s BalanceBase:%f, different from netBuy:%f. Force to use netBuy\n",
				project.Symbol, project.BalanceBase, netBuy)
			project.BalanceBase = netBuy

		}else if netBuy*highestBid.Price < 5 * MinOrderTotal {

			// that means it's already sold! project close.
			// and ignore trivial remaining balance, probably caused by Binance 'MinOrderTotal' limitation.

			fmt.Printf("ProjectManager - %s account balance=%f(%s) or %f(BTC). sold-out? then project to be close.\n",
				project.Symbol, project.AccBalanceBase, project.Symbol,
				project.AccBalanceBase*highestBid.Price)

			project.BalanceBase = netBuy
			project.IsClosed = true
		}
		project.BalanceQuote = netIncome + project.InitialBalance

		project.Roi = (project.BalanceQuote + project.BalanceBase * highestBid.Price) / project.InitialBalance - 1.0
		fmt.Printf("ProjectManager - %s: Roi=%.2f%%, RoiS=%.2f%%, LiveBalance=%f\n",
			project.Symbol, project.Roi*100, project.RoiS*100, project.AccBalanceBase*highestBid.Price)

		// Update Roi to Database
		if !UpdateProjectRoi(project){
			fmt.Println("ProjectManager - Warning! UpdateProjectRoi fail.")
		}

		// skip later part if project data is not complete yet!
		if project.Roi == 0{
			continue
		}

		// core value: auto trading!
		if !project.IsClosed {
			AutoTrading(project, true)
		}
	}

	// Remove Closed Projects from AliveProjectList
	projects := len(ActiveProjectList)
	for i:=projects-1; i>=0; i-- {

		if ActiveProjectList[i].IsClosed {
			ActiveProjectList = append(ActiveProjectList[:i], ActiveProjectList[i+1:]...)
		}
	}

	ProjectMutex.Unlock()

}

/*
	Quit Conditions:
	1、TotalLoss > 20%
	2、TotalGain > 40%
	3、Loss in latest 1 hour  				（N1HourPrice, N1HourRoi)
	4、Loss or Gain < 5% in latest 3 hours   (N3HourPrice, N1HourRoi)
	5、Loss or Gain < 5% in latest 6 hours 	 (N6HourPrice, N6HourRoi)
	6、Over 12 Hours project
	7、Manual Command：ForceQuit = True, Amount: 25%、50%、75%、100%、Default(100%)

	Others:
	1、For（1、3、4、5），add into BlackList for 2 hours
	2、QuitProtect=True，no automatic quite；only ForceQuit = True can make project quit.
 */
func QuitDecisionMake(project *ProjectData) (bool,bool){

	if project.ForceQuit {
		return true,false	//quit w/o blacklist
	}

	if project.QuitProtect {
		return false,false
	}

	if project.Roi <= -0.2{
		return true,true	//quit w/ blacklist
	}

	if project.Roi >= 0.4 {
		return true,false	//quit w/o blacklist
	}

	if project.CreateTime.Add(time.Hour * 12).Before(time.Now()) {
		return true,false	//quit w/o blacklist
	}

	roiData := GetLatestRoi(project.Symbol, 1.0)
	if roiData!=nil && roiData.RoiD < 0{
		return true,true	//quit w/ blacklist
	}

	roiData = GetLatestRoi(project.Symbol, 3.0)
	if roiData!=nil && roiData.RoiD < 0.05{
		return true,true	//quit w/ blacklist
	}

	roiData = GetLatestRoi(project.Symbol, 6.0)
	if roiData!=nil && roiData.RoiD < 0.05{
		return true,true	//quit w/ blacklist
	}

	return false,false
}

/*
 * Back Window: for example 1Hour, 3Hour, 6Hour
 */
func GetLatestRoi(symbol string, backTimeWindow float64) *RoiData{

	if backTimeWindow < 0.5 || backTimeWindow>120 {
		fmt.Println("GetLatestRoi - Error! backTimeWindow out of range [0.5,120].", backTimeWindow)
		return nil
	}

	i := 0
	var s string
	for i, s = range SymbolList {
		if symbol == s {
			break
		}
	}
	if SymbolList[i] != symbol {
		fmt.Println("GetLatestRoi - Fail to find symbol in SymbolList", symbol)
		return nil
	}

	klinesMap := SymbolKlinesMapList[i]

	var nowOpenTime= time.Time{}
	var nowCloseTime= time.Time{}
	var nowClose float64 = 0.0

	// find the latest OpenTime
	// TODO: just use Now() to get nearest 5 minutes
	for _, v := range klinesMap {
		if v.OpenTime.After(nowOpenTime) {
			nowOpenTime = v.OpenTime
			nowCloseTime = v.CloseTime
			nowClose = v.Close
		}
	}

	N := int(math.Round(backTimeWindow * 60 / 5))

	roiData := CalcRoi(symbol,
		N,
		nowOpenTime,
		nowCloseTime,
		nowClose,
		klinesMap)

	return &roiData
}

/*
 * Get Order Net Buy (total Buy - total Sell) for ProjectID, based on database records.
 *	  and Net Income (total Income - total Spent).
 */
func GetProjectNetBuy(projectId int64) (float64,float64){
	//
	//fmt.Printf("GetProjectNetBuy - func enter. ProjectID=%d\n", projectId)

	rowBuy := DBCon.QueryRow(
		"select sum(ExecutedQty),sum(ExecutedQty*Price) from order_list " +
		"where Side='BUY' and IsDone=1 and ProjectID=?;",
		projectId)

	totalExecutedBuyQty := 0.0
	totalSpentQuote := 0.0
	var t1,t2 NullFloat64
	errB := rowBuy.Scan(&t1, &t2)

	if errB != nil && errB != sql.ErrNoRows {
		level.Error(logger).Log("GetProjectSum - DB.Query Fail. Err=", errB)
		panic(errB.Error())
	}
	if t1.Valid {
		totalExecutedBuyQty = t1.Float64
	}
	if t2.Valid {
		totalSpentQuote = t2.Float64
	}

	// Sum all the Sell

	rowSell := DBCon.QueryRow(
		"select sum(ExecutedQty),sum(ExecutedQty*Price) from order_list " +
		"where Side='SELL' and IsDone=1 and ProjectID=?;",
		projectId)

	totalExecutedSellQty := 0.0
	totalIncomeQuote := 0.0

	errS := rowSell.Scan(&t1, &t2)

	if errS != nil && errS != sql.ErrNoRows {
		level.Error(logger).Log("GetProjectSum - DB.Query Fail. Err=", errS)
		panic(errS.Error())
	}

	if t1.Valid {
		totalExecutedSellQty = t1.Float64
	}
	if t2.Valid {
		totalIncomeQuote = t2.Float64
	}

	//fmt.Println("GetProjectSum - TotalBuyQty:", totalExecutedBuyQty, "TotalSellQty:", totalExecutedSellQty,
	//	". TotalSpent:", totalSpentQuote, "TotalIncome:", totalIncomeQuote,
	//	"ProjectId =", projectId)
	return totalExecutedBuyQty-totalExecutedSellQty, totalIncomeQuote-totalSpentQuote
}

/*
 * Update Project Roi into Database
 */
func UpdateProjectRoi(project *ProjectData) bool{

	if project==nil || project.id<0 {
		level.Warn(logger).Log("UpdateProjectRoi - Fail! Invalid id:", project.id)
		return false
	}

	query := `UPDATE project_list SET Roi=?, BalanceBase=?, BalanceQuote=?, IsClosed=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.Roi,
		project.BalanceBase,
		project.BalanceQuote,
		project.IsClosed,
		project.id,
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
