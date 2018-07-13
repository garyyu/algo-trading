package main

const algoSellRiseThreshold	= 0.02

/*
 * Sell Rise Algorithm
 * Description:
 *	Based on each KLine, if Close price > initialPrice, sell the gain part;
 *						 otherwise, do nothing.
 *  The idea is to keep the remain 'value' at most as initialPrice.
 */
func algoSellRise(balanceBase *float64, balanceQuote *float64,
	kline *KlineRo, initialPrice float64) (float64,float64){

	sell := 0.0
	buy := 0.0

	if initialPrice<=0{
		return 0,0
	}

	if (kline.Close - initialPrice)/initialPrice < algoSellRiseThreshold{
		return 0,0
	}

	gain := *balanceBase * (kline.Close - initialPrice)
	if gain > 0 {
		sell = gain
		if sell < MinOrderTotal {
			sell = 0
		}
	}

	*balanceQuote += sell - buy
	*balanceBase += (buy - sell)/kline.Close

	return buy,sell
}
