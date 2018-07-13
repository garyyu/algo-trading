package main

import (
	"math"
	)

func algoExample(balanceBase float64, balanceQuote float64, kline *KlineRo) (float64,float64){

	sell := 0.0
	buy := 0.0

	gain := (kline.Close - kline.Open) * balanceBase
	if gain > 0 {
		sell = math.Min(gain, kline.Close*balanceBase)
		if sell < MinOrderTotal { // note: $8 = 0.001btc on $8k/btc
			sell = 0
		}
	} else if gain < 0 {
		buy = math.Min(balanceQuote, -gain)
		if buy < MinOrderTotal {  // note: $8 = 0.001btc on $8k/btc
			buy = 0
		}
	}

	return buy,sell
}
