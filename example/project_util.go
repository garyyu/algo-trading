package main

import (
	"github.com/go-kit/kit/log/level"
	"time"
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
	AccBalanceBase			 float64	`json:"AccBalanceBase"`
	BalanceQuote			 float64	`json:"BalanceQuote"`
	Roi				 		 float64	`json:"Roi"`
	RoiS			 		 float64	`json:"RoiS"`
	InitialPrice			 float64	`json:"InitialPrice"`
	NowPrice			 	 float64	`json:"NowPrice"`
	InitialAmount			 float64	`json:"InitialAmount"`
	FeeBNB			 		 float64	`json:"FeeBNB"`
	FeeEmbed			 	 float64	`json:"FeeEmbed"`
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
		for _, existing := range ActiveProjectList {
			if existing.Symbol == hunt.Symbol {
				continue rowLoop
			}
		}

		huntList = append(huntList, hunt)
	}

	ProjectMutex.RUnlock()

	return &huntList
}

/*
 * Get active projects from local database
 */
func getActiveProjectList() int{

	rows, err := DBCon.Query(
		"select * from project_list where IsClosed=0 and CloseTime is NULL LIMIT 50")

	if err != nil {
		level.Error(logger).Log("getActiveProjectList - fail! Err=", err)
		panic(err.Error())
	}
	defer rows.Close()

rowLoop:
	for rows.Next() {

		var transactTime NullTime
		var closeTime NullTime

		project := ProjectData{}

		err := rows.Scan(&project.id, &project.Symbol, &project.ForceQuit, &project.QuitProtect,
			&project.OrderID, &project.ClientOrderID, &project.InitialBalance, &project.BalanceBase,
			&project.AccBalanceBase, &project.BalanceQuote, &project.Roi, &project.RoiS,
			&project.InitialPrice, &project.NowPrice, &project.InitialAmount, &project.FeeBNB,
			&project.FeeEmbed, &project.CreateTime, &transactTime, &project.OrderStatus,
			&closeTime, &project.IsClosed)

		if err != nil {
			level.Error(logger).Log("getActiveProjectList - Scan Err:", err)
			continue
		}

		if transactTime.Valid {
			project.TransactTime = transactTime.Time
		}
		if closeTime.Valid {
			project.CloseTime = closeTime.Time
		}

		// if already in active project list
		for _, existing := range ActiveProjectList {
			if existing.id == project.id {
				continue rowLoop
			}
		}

		ActiveProjectList = append(ActiveProjectList, &project)
	}

	return len(ActiveProjectList)
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
		level.Warn(logger).Log("InsertProject - Fail. ClientOrderID:", project.ClientOrderID)
		return -1
	}

	query := `INSERT INTO project_list (
				Symbol, ClientOrderID, InitialBalance, InitialPrice, NowPrice, 
				AccBalanceBase, InitialAmount, CreateTime, OrderStatus
			  ) VALUES (?,?,?,?,?,?,?,NOW(),?)`

	res, err := DBCon.Exec(query,
		project.Symbol,
		project.ClientOrderID,
		project.InitialBalance,
		project.InitialPrice,
		project.InitialPrice,
		project.InitialAmount,
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
 * Update Project InitialBalance into Database
 */
func UpdateProjectInitialBalance(project *ProjectData) bool{

	query := `UPDATE project_list SET InitialBalance=?, InitialPrice=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.InitialBalance,
		project.InitialPrice,
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


/*
 * Update Project OrderID into Database
 */
func UpdateProjectOrderID(project *ProjectData) bool{

	if project==nil || len(project.ClientOrderID)==0 || project.OrderID==0 || project.id<0 {
		level.Warn(logger).Log("UpdateProjectOrderID.ProjectData", project)
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


/*
 * Update Project BalanceBase into Database
 */
func UpdateProjectBalanceBase(project *ProjectData) bool{

	query := `UPDATE project_list SET BalanceBase=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.BalanceBase,
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


/*
 * Update Project Account Balance Base into Database
 *		Balance Data in Binance Account.
 */
func UpdateProjectAccBalanceBase(project *ProjectData) bool{

	query := `UPDATE project_list SET AccBalanceBase=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.AccBalanceBase,
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


/*
 * Update Fees into Database
 */
func UpdateProjectFees(project *ProjectData) bool{

	query := `UPDATE project_list SET FeeBNB=?,FeeEmbed=? WHERE id=?`

	res, err := DBCon.Exec(query,
		project.FeeBNB,
		project.FeeEmbed,
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



/*
 * Update Project IsClose into Database
 */
func UpdateProjectClose(project *ProjectData) bool{

	query := `UPDATE project_list SET IsClosed=?, CloseTime=NOW() WHERE id=?`

	res, err := DBCon.Exec(query,
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


