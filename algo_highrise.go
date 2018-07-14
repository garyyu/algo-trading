package main

const algoHighRiseThreshold1	= 0.20	// price rise ratio

/*
 * High Rise Algorithm
 * Description:
 *	Based on each KLine, if Close price > initialPrice, sell the gain part;
 *						 otherwise, do nothing.
 *  The idea is to keep the remain 'value' at most as initialPrice.
 */
func algoHighRise(balanceBase *float64,
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
	if ratio <= algoHighRiseThreshold1 {
		return 0,0
	}

	sell = *balanceBase

	*balanceQuote += (sell - buy)*kline.Close
	*balanceBase += buy - sell

	return buy,sell
}
