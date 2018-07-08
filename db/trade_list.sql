drop table if exists trade_list;

create table trade_list (
    id int primary key auto_increment,
    ProjectID	    int(11) NOT NULL DEFAULT -1,
    Symbol	    varchar(16) NOT NULL,
    TradeID	    int(11) NOT NULL DEFAULT -1,
    Price    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Qty    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Commission      DOUBLE(20,8) NOT NULL DEFAULT 0,
    CommissionAsset varchar(16) NOT NULL,
    Time            datetime DEFAULT NULL,
    IsBuyer	    tinyint NOT NULL DEFAULT 1,
    IsMaker	    tinyint NOT NULL DEFAULT 1,
    IsBestMatch	    tinyint NOT NULL DEFAULT 1,
    InsertTime      datetime DEFAULT NULL,
    UNIQUE (TradeID)
  ) comment 'trade_list table';
