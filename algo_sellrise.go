package main

import (
	"math"
)

const algoSellRiseThreshold1	= 0.05	// price rise ratio
const algoSellRiseThreshold2	= 0.25	// sell ratio on the gain
const algoSellRiseThreshold3	= 0.10	// sold-out ratio. close project if only 10% initialAmount left.

/*
 * Sell Rise Algorithm
 * Description:
 *	Based on each KLine, if Close price > initialPrice, sell the gain part;
 *						 otherwise, do nothing.
 *  The idea is to keep the remain 'value' at most as initialPrice.
 */
func algoSellRise(balanceBase *float64,
				  balanceQuote *float64,
				  kline *KlineRo,
				  initialAmount float64,
				  initialPrice float64,
				  demo bool) (float64,float64){

	sell := 0.0
	buy := 0.0

	if initialPrice<=0 {
		return 0,0
	}

	ratio := (kline.Close - initialPrice)/initialPrice
	if ratio <= algoSellRiseThreshold1 {
		return 0,0
	}

	// e^-4 = 1.8%; e^-3 = 5%; e^-2 = 13.5%; e^-1 = 36.8%; e^0 = 100%;
	logRatio := math.Log(ratio) + 1.0

	sellRatio := math.Pow(2.0, logRatio)
	//
	//if demo {
	//	fmt.Printf("algoSellRise - initialPrice=%f, nowPrice=%f. Ratio=%f, logRatio=%f, sellRatio=%f\n",
	//		initialPrice, kline.Close, ratio, logRatio, sellRatio)
	//}

	if sellRatio < algoSellRiseThreshold2{
		return 0,0
	}

	gain := *balanceBase * (kline.Close - initialPrice) * sellRatio
	if gain > 0 {

		sell = math.Min(gain/kline.Close, *balanceBase)

		// if remaining is less than 10% of initial amount, sell all out to close project.
		if *balanceBase < initialAmount * algoSellRiseThreshold3 {
			sell = *balanceBase
		}

		if sell < MinOrderTotal {
			sell = 0
		}
	}

	*balanceQuote += (sell - buy)*kline.Close
	*balanceBase += buy - sell

	return buy,sell
}
