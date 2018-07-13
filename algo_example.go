package main

import "math"

/*
 * A simple example algorithm, just for demo purpose.
 * Description:
 *	Based on each KLine, if Close price > Open price, sell the gain part;
 *						 otherwise, buy the loss part if having enough balance quote.
 *  To make the remain
 */
func algoExample(balanceBase *float64, balanceQuote *float64,
	kline *KlineRo, initialPrice float64) (float64,float64){

	sell := 0.0
	buy := 0.0

	gain := *balanceBase * (kline.Close - kline.Open)
	if gain > 0 {
		sell = math.Min(gain, *balanceBase * kline.Close)
		if sell < MinOrderTotal {
			sell = 0
		}
	} else if gain < 0 {
		buy = math.Min(*balanceQuote, -gain)
		if buy < MinOrderTotal {
			buy = 0
		}
	}

	*balanceQuote += sell - buy
	*balanceBase += (buy - sell)/kline.Close

	return buy,sell
}
