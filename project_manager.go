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

		// detection of external buy/sell behaviour (增仓/减/清仓), which will update the
		// 'InitialAmount', 'InitialPrice' and so on.
		if _,_,ok := externalTradingSum(project); !ok {
			continue
		}

		autoTradingSum(project, highestBid.Price)

		if project.InitialBalance==0{
			fmt.Printf("ProjectManager - Warning! InitialBalance=0. Roi Invalid.\n")
			project.Roi = 0
		}else {
			project.Roi = (project.BalanceQuote+project.BalanceBase*highestBid.Price)/project.InitialBalance - 1.0
		}

		fmt.Printf("ProjectManager - %s: Roi=%.2f%%, RoiS=%.2f%%, LiveBalance=%f\n",
			project.Symbol, project.Roi*100, project.RoiS*100, project.AccBalanceBase*highestBid.Price)

		// Update Roi to Database
		if !UpdateProjectRoi(project){
			fmt.Println("ProjectManager - Warning! UpdateProjectRoi fail.")
		}

		// skip later part if project data is not complete yet!
		if project.Roi == 0 || project.InitialAmount ==0 || project.InitialPrice==0 {
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

/* Detection of internal auto-trading buy/sell behaviour, which will update
 * the 'BalanceBase' and 'BalanceQuote'.
 */
func autoTradingSum(project *ProjectData, nowPrice float64){

	// sum all internal (auto-trading) orders in project
	netBuy,netIncome := GetProjectNetBuy(project.id, false)

	project.BalanceBase = netBuy + project.InitialAmount

	if project.BalanceBase*nowPrice < MinOrderTotal {

		// that means it's already sold! project close.
		// and ignore trivial remaining balance, probably caused by Binance 'MinOrderTotal' limitation.

		fmt.Printf("ProjectManager - %s account balance=%f(%s) or %f(BTC). sold-out? project close.\n",
			project.Symbol, project.AccBalanceBase, project.Symbol,
			project.AccBalanceBase*nowPrice)

		ProjectClose(project)
	}

	project.BalanceQuote = netIncome
}

/* Detection of external buy/sell behaviour (增仓/减/清仓), which will update the
 * 'InitialAmount', 'InitialPrice', 'FilledProfit' and so on.
 */
func externalTradingSum(project *ProjectData) (float64,float64,bool){

	netBuy,netIncome := GetProjectNetBuy(project.id, true)

	if netBuy == 0 && netIncome == 0 {
		fmt.Printf("ProjectManager - %s order info not finalized! wait next ticker.\n",
			project.Symbol)
		return netBuy,netIncome,false
	}else{
		//fmt.Printf("ProjectManager - %s: NetBuy=%f, NetIncome=%f\n",
		//	project.Symbol, netBuy, netIncome)
	}

	if project.InitialAmount>0 {

		if netBuy > project.InitialAmount {
			// external buy (增仓) found
			project.FilledProfit = netBuy * project.InitialPrice + netIncome

			fmt.Printf("ProjectManager - external buy(增仓) found! increased holding: %f(%s)",
				netBuy-project.InitialAmount, project.Symbol)

		} else if netBuy < project.InitialAmount {
			// external sell (减/清仓) found
			project.FilledProfit = netBuy * project.InitialPrice + netIncome

			fmt.Printf("ProjectManager - external sell(减/清仓) found! decreased holding: %f(%s)",
				project.InitialAmount-netBuy, project.Symbol)
		}

		if !FloatEquals(netBuy, project.InitialAmount){

			project.InitialAmount = netBuy
			project.InitialPrice = (project.FilledProfit-netIncome) / netBuy		// average price
			project.InitialBalance = project.InitialPrice*project.InitialAmount

			// update to database
			if !UpdateProjectInitialBalance(project){
				fmt.Println("ProjectManager - Warning! Update Project InitialAmount into database Fail. Project:",
					project.Symbol)
			}
		}
	}

	return netBuy,netIncome,true
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
	for i, s = range LivelySymbolList {
		if symbol == s {
			break
		}
	}
	if LivelySymbolList[i] != symbol {
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
		klinesMap,
		false)

	return &roiData
}

/*
 * Get Order Net Buy (total Buy - total Sell) for ProjectID, based on database records.
 *	  and Net Income (total Income - total Spent).
 *
 * getExternalOrder:
 * 	- true means external buy/sell (增仓/减仓), thanks to ClientOrderID
 *
 *	- false means internal auto-trading orders.
 *		Our 'auto_trading' order will always have a ClientOrderID with format yyyymmddhhmmssxxxx
 * 		ClientOrderID := time.Now().Format("20060102150405") + fmt.Sprintf("%04d",rand.Intn(9999))
 */
func GetProjectNetBuy(projectId int64, getExternalOrder bool) (float64,float64){
	//
	//fmt.Printf("GetProjectNetBuy - func enter. ProjectID=%d\n", projectId)

	var query string
	if getExternalOrder {
		query = "select sum(if(Side='BUY',ExecutedQty,-ExecutedQty))," +
			"sum(if(Side='BUY',-ExecutedQty*Price,ExecutedQty*Price)) from order_list " +
			"where IsDone=1 and ClientOrderID NOT REGEXP '^[0-9]{18}$' and ProjectID=?;"
	}else{
		query = "select sum(if(Side='BUY',ExecutedQty,-ExecutedQty))," +
			"sum(if(Side='BUY',-ExecutedQty*Price,ExecutedQty*Price)) from order_list " +
			"where IsDone=1 and ClientOrderID REGEXP '^[0-9]{18}$' and ProjectID=?;"
	}

	rowBuy := DBCon.QueryRow(query, projectId)

	netBuy := 0.0
	netIncome := 0.0
	var t1,t2 NullFloat64
	errB := rowBuy.Scan(&t1, &t2)

	if errB != nil && errB != sql.ErrNoRows {
		level.Error(logger).Log("GetProjectSum - DB.Query Fail. Err=", errB)
		panic(errB.Error())
	}
	if t1.Valid {
		netBuy = t1.Float64
	}
	if t2.Valid {
		netIncome = t2.Float64
	}

	return netBuy, netIncome
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
