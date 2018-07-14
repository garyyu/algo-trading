package main

import "math"

/*
 * A simple example algorithm, just for demo purpose.
 * Description:
 *	Based on each KLine, if Close price > Open price, sell the gain part;
 *						 otherwise, buy the loss part if having enough balance quote.
 *  To make the remain
 */
func algoExample(balanceBase *float64,
				 balanceQuote *float64,
				 kline *KlineRo,
				 initialAmount float64,
				 initialPrice float64,
				 demo bool) (float64,float64){

	sell := 0.0
	buy := 0.0

	gain := *balanceBase * (kline.Close - kline.Open) / kline.Close
	if gain > 0 {
		sell = math.Min(gain, *balanceBase)
		if sell*kline.Close < MinOrderTotal {
			sell = 0
		}
	} else if gain < 0 {
		buy = math.Min(*balanceQuote/kline.Close, -gain)
		if buy*kline.Close < MinOrderTotal {
			buy = 0
		}
	}

	*balanceQuote += (sell - buy)*kline.Close
	*balanceBase += buy - sell

	return buy,sell
}
