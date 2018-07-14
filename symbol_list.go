package main

var SymbolList = [...]string{

	/* Volume: $[10M - 57M] */

	"ETHBTC",
	"EOSBTC",
	"KEYBTC",
	"ETCBTC",
	"TRXBTC",
	"POEBTC",
	"NEOBTC",
	"ONTBTC",
	"XRPBTC",
	"ADABTC",
	"ICXBTC",
	"QKCBTC",
	"BCDBTC",
	"BCCBTC",
	//"BCNBTC",		// remove this boring one
	"BNBBTC",
	"XLMBTC",
	"LTCBTC",
	"ZRXBTC",
	"VENBTC",
	"ZILBTC",
	"NCASHBTC",
	"IOTABTC",
	"NPXSBTC",

	/* Volume: $[5M - 10M] */

	"GTOBTC",
	"XVGBTC",
	"CMTBTC",
	"NANOBTC",
	"ARNBTC",
	"THETABTC",
	"EVXBTC",
	"WANBTC",
	"TUSDBTC",
	"XMRBTC",
	"IOTXBTC",

	/* Volume: $[2M - 5M] */

	"WAVESBTC",
	"NASBTC",
	"SNTBTC",
	"LOOMBTC",
	"DASHBTC",
	"BQXBTC",
	"STORMBTC",
	"CLOAKBTC",
	"IOSTBTC",
	"LUNBTC",
	"XEMBTC",
	"OMGBTC",
	"ELFBTC",
	"QTUMBTC",
	"NAVBTC",
	"ENGBTC",
	"GVTBTC",
	"QLCBTC",
	"POABTC",
	"STRATBTC",
	"SKYBTC",
	"RCNBTC",
	"POWRBTC",
	"WTCBTC",
	"SCBTC",
	"AEBTC",

	/* Volume: $[1.3M - 2M] */

	"BRDBTC",
	"ZECBTC",
	"EDOBTC",
	"AIONBTC",
	"SUBBTC",
	"DNTBTC",
	"TNBBTC",
	"STEEMBTC",
	"BTSBTC",
	"BCPTBTC",
	"INSBTC",
	"NXSBTC",
	"LENDBTC",
	"OSTBTC",
	"BTGBTC",
}

type Precision struct {
	PricePrecision int
	AmountPrecision int
}

var SymbolPrecision = map[string]Precision{

	/* Volume: $[10M - 57M] */

	"ETHBTC": {6,3},
	"EOSBTC": {7,2},
	"KEYBTC": {8,0},
	"ETCBTC": {6,2},
	"TRXBTC": {8,0},
	"POEBTC": {8,0},
	"NEOBTC": {6,2},
	"ONTBTC": {7,2},
	"XRPBTC": {8,0},
	"ADABTC": {8,0},
	"ICXBTC": {7,2},
	"QKCBTC": {8,0},
	"BCDBTC": {6,3},
	"BCCBTC": {6,3},
	"BCNBTC": {8,0},
	"BNBBTC": {7,2},
	"XLMBTC": {8,0},
	"LTCBTC": {6,2},
	"ZRXBTC": {8,0},
	"VENBTC": {8,0},
	"ZILBTC": {8,0},
	"NCASHBTC": {8,0},
	"IOTABTC": {8,0},
	"NPXSBTC": {8,0},

	/* Volume: $[5M - 10M] */

	"GTOBTC": {8,0},
	"XVGBTC": {8,0},
	"CMTBTC": {8,0},
	"NANOBTC": {7,2},
	"ARNBTC": {8,0},
	"THETABTC": {8,0},
	"EVXBTC": {8,0},
	"WANBTC": {7,2},
	"TUSDBTC": {8,0},
	"XMRBTC": {6,3},
	"IOTXBTC": {8,0},

	/* Volume: $[2M - 5M] */

	"WAVESBTC": {7,2},
	"NASBTC": {7,2},
	"SNTBTC": {8,0},
	"LOOMBTC": {8,0},
	"DASHBTC": {6,3},
	"BQXBTC": {8,0},
	"STORMBTC": {8,0},
	"CLOAKBTC": {7,2},
	"IOSTBTC": {8,0},
	"LUNBTC": {7,2},
	"XEMBTC": {8,0},
	"OMGBTC": {6,2},
	"ELFBTC": {8,0},
	"QTUMBTC": {6,2},
	"NAVBTC": {7,2},
	"ENGBTC": {8,0},
	"GVTBTC": {7,2},
	"QLCBTC": {8,0},
	"POABTC": {8,0},
	"STRATBTC": {7,2},
	"SKYBTC": {6,3},
	"RCNBTC": {8,0},
	"POWRBTC": {8,0},
	"WTCBTC": {7,2},
	"SCBTC": {8,0},
	"AEBTC": {7,2},

	/* Volume: $[1.3M - 2M] */

	"BRDBTC": {8,0},
	"ZECBTC": {6,3},
	"EDOBTC": {7,2},
	"AIONBTC": {7,2},
	"SUBBTC": {8,0},
	"DNTBTC": {8,0},
	"TNBBTC": {8,0},
	"STEEMBTC": {7,2},
	"BTSBTC": {8,0},
	"BCPTBTC": {8,0},
	"INSBTC": {7,2},
	"NXSBTC": {7,2},
	"LENDBTC": {8,0},
	"OSTBTC": {8,0},
	"BTGBTC": {6,2},
}
