drop table if exists project_list;

create table project_list (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    ForceQuit	    tinyint NOT NULL DEFAULT 0, 
    QuitProtect	    tinyint NOT NULL DEFAULT 0, 
    OrderID	    int(11) NOT NULL DEFAULT -1,
    ClientOrderID   varchar(18) NOT NULL,
    InitialBalance  DOUBLE(20,8) NOT NULL DEFAULT 0,
    BalanceBase     DOUBLE(20,8) NOT NULL DEFAULT 0,
    BalanceQuote    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Roi     	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    RoiS    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    InitialPrice    DOUBLE(20,8) NOT NULL DEFAULT 0,
    InitialAmount   DOUBLE(20,8) NOT NULL DEFAULT 0,
    CreateTime      datetime NOT NULL,
    TransactTime    datetime DEFAULT NULL,
    CloseTime       datetime DEFAULT NULL,
    IsClosed	    tinyint NOT NULL DEFAULT 0, 
    UNIQUE (ClientOrderID)
  ) comment 'project_list table';
