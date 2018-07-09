package main

import (
	"fmt"
	"time"
)

/*
 *  Main Routine for Project Management
 */
func ProjectRoutine(){

	fmt.Printf("ProjectTick Start: \t%s\n\n", time.Now().Format("2006-01-02 15:04:05.004005683"))

	// start a goroutine to get realtime project management in 1 min interval
	ticker := projectTicker()
	var tickerCount = 0
loop:
	for  {
		select {
		case _ = <- routinesExitChan:
			break loop

		case tick := <-ticker.C:
			ticker.Stop()

			tickerCount += 1
			fmt.Printf("ProjectTick: \t\t%s\t%d\n", tick.Format("2006-01-02 15:04:05.004005683"), tickerCount)

			// account query can auto-import the project and tracking
			// todo: this can be in lower interval for example 5 minutes, since it's only used for new project import
			QueryAccount()

			// trades query can give the new project basic info such as InitialBalance, InitialPrice.
			QueryMyTrades()

			// orders query can get the final state of order if it's not finalized,
			// but they are already in local database via GetAllOrders() in ProjectManager()
			QueryOrders()

			ProjectManager()

			RoiReport()

			// Update the ticker
			ticker = projectTicker()

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println("goroutine exited - ProjectRoutine")
}

func projectTicker() *time.Ticker {

	now := time.Now()
	second := 45 - now.Second()		// range: [45..-15]
	if second <= 0 {
		second += 60				// range: [45..0] + [45..60] -> [0..60]
	}

	return time.NewTicker(
		time.Second * time.Duration(second) -
			time.Nanosecond * time.Duration(now.Nanosecond()))
}

