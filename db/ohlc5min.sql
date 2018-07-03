drop table if exists ohlc5min;


create table ohlc5min (
    id int primary key auto_increment,
    Symbol	varchar(16) NOT NULL,
    OpenTime  	datetime NOT NULL,
    Open      	DOUBLE(20,8) NOT NULL DEFAULT 0,
    High      	DOUBLE(20,8) NOT NULL DEFAULT 0,
    Low     	DOUBLE(20,8) NOT NULL DEFAULT 0,
    Close   	DOUBLE(20,8) NOT NULL DEFAULT 0,
    Volume	DOUBLE(20,8) NOT NULL DEFAULT 0,	
    CloseTime 	datetime NOT NULL,
    QuoteAssetVolume	DOUBLE(20,8) NOT NULL DEFAULT 0,
    NumberOfTrades	int(12) NOT NULL DEFAULT 0,
    TakerBuyBaseAssetVolume	DOUBLE(20,8) NOT NULL DEFAULT 0,
    TakerBuyQuoteAssetVolume	DOUBLE(20,8) NOT NULL DEFAULT 0,
    exchangeName varchar(16) NOT NULL,
    insertTime   datetime NOT NULL,
    updateTime   datetime
  ) comment 'ohlc5min table';
