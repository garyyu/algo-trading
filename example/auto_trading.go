package main

import (
	"time"
	"math/rand"
	"sync"
	"fmt"
	"bitbucket.org/garyyu/go-binance"
	"github.com/go-kit/kit/log/level"
	"math"
)

type TimePrice struct {
	TradeTime time.Time
	Price float64
}

var (
	ProjectMutex sync.RWMutex
	ActiveProjectList []*ProjectData
	globalBalanceQuote	float64

	PrevAutoTradingList = make(map[string]TimePrice)
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

/*
 * Auto Trading for Project
 */
func AutoTrading(project *ProjectData, dryrun bool) bool{

	_,ok := PrevAutoTradingList[project.Symbol]
	if !ok {

		timePrice := GetTimePrice(project.Symbol)
		if timePrice.TradeTime.IsZero() {
			fmt.Println("Error! AutoTrading - GetTimePrice Fail. skip auto trading", project.Symbol)
			return false
		}

		PrevAutoTradingList[project.Symbol] = timePrice
	}

	// get latest price
	highestBid := OBData{}
	highestBid = getHighestBid(project.Symbol)
	if highestBid.Time.Add(time.Second * 5).Before(time.Now()) {
		fmt.Println("Warning! AutoTrading - getHighestBid got old data. skip auto trading", project.Symbol)
		return false
	}

	// last auto-trading already past 5 minutes?
	if PrevAutoTradingList[project.Symbol].TradeTime.Add(time.Minute * 1).After(highestBid.Time) {
		return false
	}

	// check if need auto sell/buy

	var sell float64 = 0.0
	var buy  float64 = 0.0
	gain := (highestBid.Price - PrevAutoTradingList[project.Symbol].Price) * project.BalanceBase

	//duration := highestBid.Time.Sub(PrevAutoTradingList[project.Symbol].TradeTime)

	if gain > 0 {
		sell = gain

		if sell < 1.2*MinOrderTotal { // note: $8 = 0.001btc on $8k/btc

			fmt.Printf("AutoTrading - %s Trivial Sell Request %.8f Ignored. Price=%f->%f\n",
				project.Symbol, sell, PrevAutoTradingList[project.Symbol].Price, highestBid.Price)
			sell = 0.0
		}
	} else if gain < 0 {
		buy = math.Min(project.BalanceQuote, -gain)

		if buy < 1.2*MinOrderTotal {  // note: $8 = 0.001btc on $8k/btc

			fmt.Printf("AutoTrading - %s Trivial Buy Request %.8f Ignored. Price=%f->%f\n",
				project.Symbol, buy, PrevAutoTradingList[project.Symbol].Price, highestBid.Price)
			buy = 0.0
		}
	}

	if sell<=0 && buy <=0{
		return false
	}

	// Binance limit different precision for each asset
	precision, ok := SymbolPrecision[project.Symbol]
	if !ok{
		level.Error(logger).Log("Symbol", project.Symbol, "missing precision data!")
		return false
	}

	// Round to Binance Precision
	var direction binance.OrderSide
	if buy > 0 {
		direction = binance.SideBuy
	}else{
		direction = binance.SideSell
	}
	amount := math.Max(buy, sell)

	quantity := Round(amount/highestBid.Price, math.Pow10(precision.AmountPrecision))
	price := Round(highestBid.Price, math.Pow10(precision.PricePrecision))

	ClientOrderID := time.Now().Format("20060102150405") + fmt.Sprintf("%04d",rand.Intn(9999))

	// call binance API to make new order
	var OrderID int64 = -1
	if !dryrun {
		newOrder, err := binanceSrv.NewOrder(binance.NewOrderRequest{
			Symbol:           project.Symbol,
			Quantity:         quantity,
			Price:            price,
			NewClientOrderID: ClientOrderID,
			Side:             direction,
			TimeInForce:      binance.GTC,
			Type:             binance.TypeLimit,
			Timestamp:        time.Now(),
		})
		if err != nil {
			level.Error(logger).Log("AutoTrading - NewOrder Fail. Err:", err, "Project:", project.Symbol)
			return false
		}
		OrderID = newOrder.OrderID
	}

	fmt.Printf("AutoTrading - New %s OrderID: %d. Price:%f, Amount:%f. Time:%s\n",
		string(direction), OrderID, price, amount, time.Now().Format("20060102150405"))

	// update prevAutoTrading
	PrevAutoTradingList[project.Symbol] = TimePrice{
		TradeTime: 	time.Now().Local(),
		Price:		price,
	}

	// insert data into order list
	orderData := OrderData{
		ProjectID: project.id,
		executedOrder:binance.ExecutedOrder{
			Symbol: project.Symbol,
			OrderID: OrderID,
			ClientOrderID: ClientOrderID,
			Price: price,
			OrigQty: quantity,
		},
	}
	if !dryrun {
		InsertOrder(&orderData)
	}

	return true
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
