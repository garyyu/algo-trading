drop table if exists roi5min;


create table roi5min (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Rank	    int(12) NOT NULL DEFAULT 0,
    InvestPeriod    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Klines	    int(12) NOT NULL DEFAULT 0,
    RoiD      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    RoiS      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    QuoteAssetVolume	DOUBLE(20,8) NOT NULL DEFAULT 0,
    NumberOfTrades  int(12) NOT NULL DEFAULT 0,
    OpenTime 	    datetime NOT NULL,
    EndTime 	    datetime NOT NULL,
    AnalysisTime    datetime NOT NULL 
  ) comment 'roi5min table';
