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

type ProjectData struct {
	id				 		 int64		`json:"id"`
	Symbol            		 string 	`json:"Symbol"`
	ForceQuit			 	 bool		`json:"ForceQuit"`
	QuitProtect 			 bool		`json:"QuitProtect"`
	OrderID					 int64		`json:"OrderID"`
	ClientOrderID 		     string 	`json:"ClientOrderID"`
	InitialBalance			 float64	`json:"InitialBalance"`
	BalanceBase				 float64	`json:"BalanceBase"`
	BalanceQuote			 float64	`json:"BalanceQuote"`
	Roi				 		 float64	`json:"Roi"`
	RoiS			 		 float64	`json:"RoiS"`
	InitialPrice			 float64	`json:"InitialPrice"`
	NowPrice			 	 float64	`json:"NowPrice"`
	InitialAmount			 float64	`json:"InitialAmount"`
	CreateTime               time.Time	`json:"CreateTime"`
	TransactTime             time.Time	`json:"TransactTime"`
	OrderStatus				 string 	`json:"OrderStatus"`
	CloseTime                time.Time	`json:"CloseTime"`
	IsClosed				 bool		`json:"IsClosed"`
}

type HuntList struct {
	id				 int64		`json:"id"`
	Symbol           string 	`json:"Symbol"`
	ForceEnter		 bool		`json:"ForceEnter"`
	Amount			 float64	`json:"Amount"`
	Time             time.Time	`json:"Time"`
}

type BlacklistHunt struct {
	id				 int64		`json:"id"`
	Symbol           string 	`json:"Symbol"`
	Reason           string 	`json:"Reason"`
	Time             time.Time	`json:"Time"`
}


var (
	ProjectMutex sync.RWMutex
	AliveProjectList []ProjectData
	globalBalanceQuote	float64
)


const MaxTradeList = 2	//12
const MinOrderTotal = 0.001		// $8 = 0.001btc on $8k/btc

func ProjectTrackIni(){

	AliveProjectList = make([]ProjectData, 0)
	// update active project list
	getAliveProjectList()

	globalBalanceQuote = 0.0005
	rand.Seed(time.Now().UTC().UnixNano())
}

func ProjectNew(){

	CancelOpenOrders()

	//fmt.Println("ProjectNew func enter")

	ProjectMutex.RLock()
	aliveProjects := len(AliveProjectList)
	ProjectMutex.RUnlock()

	// skip if already full, or run out of cash. note: $8 = 0.001btc on $8k/btc
	if aliveProjects >= MaxTradeList || globalBalanceQuote < MinOrderTotal {
		fmt.Println("ProjectNew - skip. full or run out of cash. aliveProjects=",
			aliveProjects, "globalBalanceQuote=", globalBalanceQuote)
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
		InitialBalance := globalBalanceQuote / float64(MaxTradeList-aliveProjects)

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
		aliveProjects += 1

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

func getHuntList() *[]HuntList{

	huntList := make([]HuntList, 0)

	rows, err := DBCon.Query(
		"select * from hunt_list where Time > DATE_SUB(NOW(),INTERVAL 5 MINUTE) " +
			  "order by id desc limit 32")

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	hunt := HuntList{id: -1}

	ProjectMutex.RLock()

	rowLoop:
	for rows.Next() {
		err := rows.Scan(&hunt.id, &hunt.Symbol, &hunt.ForceEnter, &hunt.Amount, &hunt.Time)
		if err != nil {
			level.Error(logger).Log("getHuntList.err", err)
			break
		}

		// if duplicate one
		for _, existing := range huntList {
			if existing.Symbol == hunt.Symbol {
				continue rowLoop
			}
		}

		// if already in active project list
		for _, existing := range AliveProjectList {
			if existing.Symbol == hunt.Symbol {
				continue rowLoop
			}
		}

		huntList = append(huntList, hunt)
	}

	ProjectMutex.RUnlock()

	return &huntList
}

//func getAliveProjects() int{
//
//	rows, err := DBCon.Query(
//		"select count(id) from project_list where IsClosed=0 and CloseTime is not NULL")
//
//	if err != nil {
//		panic(err.Error()) // proper error handling instead of panic in your app
//	}
//	defer rows.Close()
//
//	var aliveProjects = -1	// if not found, rows is empty.
//	for rows.Next() {
//		err := rows.Scan(&aliveProjects)
//		if err != nil {
//			level.Error(logger).Log("getAliveProjects.err", err)
//		}
//	}
//	return aliveProjects
//}

func getAliveProjectList() int{

	rows, err := DBCon.Query(
		"select * from project_list where IsClosed=0 and CloseTime is NULL LIMIT 50")

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	rowLoop:
	for rows.Next() {

		var transactTime NullTime
		var closeTime NullTime

		project := ProjectData{}

		err := rows.Scan(&project.id, &project.Symbol, &project.ForceQuit, &project.QuitProtect,
			&project.OrderID, &project.ClientOrderID, &project.InitialBalance, &project.BalanceBase,
			&project.BalanceQuote, &project.Roi, &project.RoiS, &project.InitialPrice,
			&project.NowPrice, &project.InitialAmount, &project.CreateTime, &transactTime,
			&project.OrderStatus, &closeTime, &project.IsClosed)

		if err != nil {
			level.Error(logger).Log("getAliveProjectList.err", err)
		}

		if transactTime.Valid {
			project.TransactTime = transactTime.Time
		}
		if closeTime.Valid {
			project.CloseTime = closeTime.Time
		}

		// if already in active project list
		for _, existing := range AliveProjectList {
			if existing.id == project.id {
				continue rowLoop
			}
		}

		AliveProjectList = append(AliveProjectList, project)
	}

	return len(AliveProjectList)
}

func getBlackList() *[]BlacklistHunt{

	blackList := make([]BlacklistHunt, 0)

	rows, err := DBCon.Query(
		"select * from blacklist_hunt where Time > DATE_SUB(NOW(),INTERVAL 2 HOUR) " +
			"order by id desc")

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer rows.Close()

	black := BlacklistHunt{}

	for rows.Next() {
		err := rows.Scan(&black.id, &black.Symbol, &black.Reason, &black.Time)
		if err != nil {
			level.Error(logger).Log("getBlackList.err", err)
		}

		blackList = append(blackList, black)
	}
	return &blackList
}

/*
 * Insert Project data into Database
 */
func InsertProject(project *ProjectData) int64{

	if project==nil || len(project.ClientOrderID)==0 {
		level.Warn(logger).Log("InsertProject.ProjectData", project)
		return -1
	}

	query := `INSERT INTO project_list (
				Symbol, ClientOrderID, InitialBalance, InitialPrice, NowPrice, 
				InitialAmount, CreateTime, OrderStatus
			  ) VALUES (?,?,?,?,?,?,NOW(),?)`

	res, err := DBCon.Exec(query,
			project.Symbol,
			project.ClientOrderID,
			project.InitialBalance,
			project.InitialPrice,
			project.InitialPrice,
			project.InitialAmount,
			project.OrderStatus,
	)

	if err != nil {
		level.Error(logger).Log("DBCon.Exec", err)
		return -1
	}

	id, _ := res.LastInsertId()
	return id
}


/*
 * Update Project OrderID into Database
 */
func UpdateProjectOrderID(project *ProjectData) bool{

	if project==nil || len(project.ClientOrderID)==0 || project.OrderID==0 || project.id<0 {
		level.Warn(logger).Log("InsertProject.ProjectData", project)
		return false
	}

	query := `UPDATE project_list SET OrderID=?, TransactTime=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.OrderID,
		project.TransactTime,
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
