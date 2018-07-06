package main

import (
	"fmt"
	"github.com/go-kit/kit/log/level"
	"database/sql"
	"time"
)

/*
 * Robot Project Manager
 */
func ProjectManager(){

	// Project Performance Index Refreshing
	ProjectMutex.Lock()
	for _, project := range AliveProjectList {

		netBuy,netIncome := GetProjectNetBuy(project.id)

		project.BalanceBase = netBuy
		project.BalanceQuote = netIncome + project.InitialBalance

		// get latest price
		highestBid := OBData{}
		highestBid = getHighestBid(project.Symbol)
		if highestBid.Time.Add(time.Second * 60).Before(time.Now()) {
			fmt.Println("Warning! ProjectManager - getHighestBid got old data. fail to update Roi for", project.Symbol)
			continue
		}

		project.Roi = (project.BalanceQuote + project.BalanceBase * highestBid.Price) / project.InitialBalance - 1.0
		fmt.Printf("ProjectManager - %s: Roi=%.2f%%, RoiS=%.2f%%\n",
			project.Symbol, project.Roi*100, project.RoiS*100)

		// Update Roi to Database
		if !UpdateProjectRoi(&project){
			fmt.Println("ProjectManager - Warning! UpdateProjectRoi fail.")
		}
	}
	ProjectMutex.Unlock()

}

/*
 * Get Order Net Buy (total Buy - total Sell) for ProjectID, based on database records.
 *	  and Net Income (total Income - total Spent).
 */
func GetProjectNetBuy(projectId int64) (float64,float64){

	// Sum all the Buy

	rowBuy := DBCon.QueryRow(
		"select sum(ExecutedQty),sum(ExecutedQty*Price) from order_list " +
		"where Side='BUY' and Status='FILLED' and IsDone=1 and ProjectID=?;",
		projectId)

	totalExecutedBuyQty := 0.0
	totalSpentQuote := 0.0
	errB := rowBuy.Scan(&totalExecutedBuyQty, &totalSpentQuote)

	if errB != nil && errB != sql.ErrNoRows {
		level.Error(logger).Log("GetProjectSum - DB.Query Fail. Err=", errB)
		panic(errB.Error())
	}

	// Sum all the Sell

	rowSell := DBCon.QueryRow(
		"select sum(ExecutedQty),sum(ExecutedQty*Price) from order_list " +
		"where Side='SELL' and Status='FILLED' and IsDone=1 and ProjectID=?;",
		projectId)

	totalExecutedSellQty := 0.0
	totalIncomeQuote := 0.0

	var t1,t2 NullFloat64
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

	if project==nil || len(project.ClientOrderID)==0 || project.OrderID==0 || project.id<0 {
		level.Warn(logger).Log("UpdateProjectRoi.ProjectData", project)
		return false
	}

	query := `UPDATE project_list SET Roi=?, BalanceBase=?, BalanceQuote=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.Roi,
		project.BalanceBase,
		project.BalanceQuote,
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
