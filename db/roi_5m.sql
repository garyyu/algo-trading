drop table if exists roi_5m;


create table roi_5m (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    RoiRank	    int(12) NOT NULL DEFAULT 0,
    InvestPeriod    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Klines	    int(12) NOT NULL DEFAULT 0,
    RoiD      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    RoiS      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    QuoteAssetVolume	DOUBLE(20,8) NOT NULL DEFAULT 0,
    NumberOfTrades  int(12) NOT NULL DEFAULT 0,
    OpenTime 	    datetime NOT NULL,
    EndTime 	    datetime NOT NULL,
    AnalysisTime    datetime NOT NULL 
  ) comment 'roi_5m table';
