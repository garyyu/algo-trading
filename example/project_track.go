package main

import (
	"time"
	"github.com/go-kit/kit/log/level"
	"math/rand"
	"bitbucket.org/garyyu/go-binance"
	"fmt"
	"math"
	"sync"
)


var (
	ProjectMutex sync.RWMutex
	ActiveProjectList []*ProjectData
	globalBalanceQuote	float64
)


const MaxTradeList = 2	//12
const MinOrderTotal = 0.001		// $8 = 0.001btc on $8k/btc

func ProjectTrackIni(){

	ActiveProjectList = make([]*ProjectData, 0)
	// update active project list
	getActiveProjectList()

	globalBalanceQuote = 0.0005
	rand.Seed(time.Now().UTC().UnixNano())
}

func ProjectNew(){

	CancelOpenOrders()

	//fmt.Println("ProjectNew func enter")

	ProjectMutex.RLock()
	activeProjects := len(ActiveProjectList)
	ProjectMutex.RUnlock()

	// skip if already full, or run out of cash. note: $8 = 0.001btc on $8k/btc
	if activeProjects >= MaxTradeList || globalBalanceQuote < MinOrderTotal {
		fmt.Println("ProjectNew - skip. full or run out of cash. activeProjects=",
			activeProjects, "globalBalanceQuote=", globalBalanceQuote)
		return
	}

	huntList := getHuntList()

	blackList := getBlackList()

	huntLoop:
	for _, hunt := range *huntList {

		// if it's a blacklist asset, skip it
		for _, black := range *blackList {
			if black.Symbol == hunt.Symbol {
				continue huntLoop
			}
		}

		ClientOrderID := time.Now().Format("20060102150405") + fmt.Sprintf("%04d",rand.Intn(9999))
		InitialBalance := globalBalanceQuote / float64(MaxTradeList-activeProjects)

		// create new project with hunt.Symbol
		NewProject := ProjectData{
			id:-1,
			Symbol:hunt.Symbol,
			ClientOrderID: ClientOrderID,
			OrderStatus: string(binance.StatusNew),
		}

		// get latest price
		highestBid := OBData{}
		for i:=0; ;i++{
			highestBid = getHighestBid(NewProject.Symbol)
			if highestBid.Time.Add(time.Second * 5).After(time.Now()) {
				break
			}

			switch i{
			case 0,1:
				time.Sleep(1 * time.Second)
			case 2:
				time.Sleep(3 * time.Second)
			case 3:
				time.Sleep(5 * time.Second)
			default:
				fmt.Println("warning! getHighestBid always got old data. fail to new a project")
				return
			}
		}

		// double check the price
		if highestBid.Symbol != NewProject.Symbol || highestBid.Quantity <= 0{
			level.Error(logger).Log("Symbol", NewProject.Symbol, "highestBid data exception!", highestBid)
			continue
		}

		// Binance limit different precision for each asset
		precision, ok := SymbolPrecision[NewProject.Symbol]
		if !ok{
			level.Error(logger).Log("Symbol", NewProject.Symbol, "missing precision data!")
			continue
		}

		quantity := Round(InitialBalance/highestBid.Price, math.Pow10(precision.AmountPrecision))
		price := Round(highestBid.Price, math.Pow10(precision.PricePrecision))

		// adjust initial balance according to precision limitation
		InitialBalance = quantity * price
		NewProject.InitialBalance = InitialBalance

		NewProject.InitialPrice = price
		NewProject.InitialAmount = quantity

		if InitialBalance < MinOrderTotal {
			level.Warn(logger).Log("globalBalanceQuote", globalBalanceQuote,
				"balance low! can't create new project for symbol:", NewProject.Symbol)
			continue
		}

		// save into database
		id := InsertProject(&NewProject)
		if id<0 {
			level.Error(logger).Log("Error! InsertProject fail. NewProject=", NewProject)
			return
		}
		NewProject.id = id

		// call binance API to make new order
		newOrder, err := binanceSrv.NewOrder(binance.NewOrderRequest{
			Symbol:      NewProject.Symbol,
			Quantity:    quantity,
			Price:       price,
			NewClientOrderID:	NewProject.ClientOrderID,
			Side:        binance.SideBuy,
			TimeInForce: binance.GTC,
			Type:        binance.TypeLimit,
			Timestamp:   time.Now(),
		})
		if err != nil {
			level.Error(logger).Log("ProjectNew - fail. error=", err, "NewProject=", NewProject)
			panic(err)
		}
		fmt.Println("ProjectNew - New Order:", newOrder)

		// update remaining balance
		globalBalanceQuote -= InitialBalance

		// update alive projects
		activeProjects += 1

		// update data into database
		NewProject.OrderID = newOrder.OrderID
		NewProject.TransactTime  = newOrder.TransactTime

		if !UpdateProjectOrderID(&NewProject) {
			level.Error(logger).Log("ProjectNew - database update fail! NewProject=", NewProject)
		}

		// insert data into order list
		orderData := OrderData{
			ProjectID: NewProject.id,
			executedOrder:binance.ExecutedOrder{
				Symbol: NewProject.Symbol,
				OrderID: NewProject.OrderID,
				ClientOrderID: NewProject.ClientOrderID,
				Price: NewProject.InitialPrice,
				OrigQty: NewProject.InitialAmount,
			},
		}
		InsertOrder(&orderData)
	}

	//fmt.Println("ProjectNew func left")
}
