drop table if exists ob_binance;


create table ob_binance (
    id int primary key auto_increment,
    LastUpdateID    int(12) NOT NULL DEFAULT 0,
    Symbol	    varchar(16) NOT NULL,
    Type  	    ENUM('Bid', 'Ask', 'NA') NOT NULL DEFAULT 'NA',
    Price      	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Quantity   	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Time    	    datetime NOT NULL 
  ) comment 'ob_binance table';
