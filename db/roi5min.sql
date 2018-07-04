drop table if exists roi5min;


create table roi5min (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Rank	    int(12) NOT NULL DEFAULT 0,
    InvestPeriod    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Klines	    int(12) NOT NULL DEFAULT 0,
    Roi      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    EndTime 	    datetime NOT NULL,
    TickerCount	    int(12) NOT NULL DEFAULT 0
  ) comment 'roi5min table';
