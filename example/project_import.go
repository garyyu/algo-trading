package main

import (
	"github.com/go-kit/kit/log/level"
	"fmt"
	"bitbucket.org/garyyu/go-binance"
	"time"
	"math/rand"
)

type TradeData struct {
	id				int64	`json:"id"`
	ProjectID		int64	`json:"ProjectID"`
	Symbol          string 	`json:"Symbol"`
	TradeID         int64	`json:"TradeID"`
	Price           float64	`json:"Price"`
	Qty             float64	`json:"Qty"`
	Commission      float64	`json:"Commission"`
	CommissionAsset string	`json:"CommissionAsset"`
	Time            time.Time	`json:"Time"`
	IsBuyer         bool	`json:"IsBuyer"`
	IsMaker         bool	`json:"IsMaker"`
	IsBestMatch     bool	`json:"IsBestMatch"`
	InsertTime		time.Time	`json:"InsertTime"`
}

/*
 * Expensive API: Weight=5
 */
func QueryAccount() {

	accountInfo, err := binanceSrv.Account(binance.AccountRequest{
		RecvWindow: 5 * time.Second,
		Timestamp:  time.Now(),
	})
	if err != nil {
		level.Error(logger).Log("QueryAccount - fail! Err=", err)
		return
	}

	lookForNew:
	for _, balance := range accountInfo.Balances {

		if balance.Asset == "BTC" || balance.Asset == "ETH" || balance.Free == 0{
			continue
		}

		asset := balance.Asset + "BTC"

		// get latest price
		highestBid := getHighestBid(asset)
		if highestBid.Time.Add(time.Second * 60).Before(time.Now()) {
			fmt.Println("Warning! QueryAccount - getHighestBid got old data. fail to manage its project", asset)
			continue
		}

		// ignore trivial balance
		if highestBid.Price * balance.Free < 5 * MinOrderTotal {
			continue
		}

		for _, knownProject := range ActiveProjectList {

			// Existing Known Project?
			if knownProject.Symbol == asset {

				if !FloatEquals(knownProject.InitialAmount, balance.Free) {
					fmt.Println("QueryAccount - Info: found new balance for",
						knownProject.Symbol, "new=",balance.Free,
						"old=", knownProject.InitialAmount)
				}

				continue lookForNew
			}
		}

		// Must Be a New Project!
		ProjectImport(balance)
	}
}

func ProjectClose(project *ProjectData){

	project.IsClosed = true

	if !UpdateProjectClose(project) {
		level.Error(logger).Log("ProjectClose - database update fail! project=", project)
	}else{
		fmt.Println("ProjectClose - Done. Project Info:", project)
	}
}

func ProjectImport(balance *binance.Balance){

	ClientOrderID := time.Now().Format("20060102150405") + fmt.Sprintf("%04d",rand.Intn(9999))

	asset := balance.Asset + "BTC"

	// create new project with hunt.Symbol
	NewProject := ProjectData{
		id:-1,
		Symbol:asset,
		ClientOrderID: ClientOrderID,
		InitialAmount: balance.Free,
		BalanceBase: balance.Free,
		OrderStatus: string(binance.StatusNew),
	}

	// save into database
	id := InsertProject(&NewProject)
	if id<0 {
		level.Error(logger).Log("Error! InsertProject fail. NewProject=", NewProject)
		return
	}
	NewProject.id = id

	// add it into AliveProjectList
	ProjectMutex.Lock()
	ActiveProjectList = append(ActiveProjectList, &NewProject)
	ProjectMutex.Unlock()

}
