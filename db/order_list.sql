drop table if exists order_list;

create table order_list (
    id int primary key auto_increment,
    ProjectID	    int(11) NOT NULL,
    IsDone	    tinyint NOT NULL DEFAULT 0, 
    Symbol	    varchar(16) NOT NULL,
    OrderID	    int(11) NOT NULL DEFAULT -1,
    ClientOrderID   varchar(32) NOT NULL,
    Price    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    OrigQty    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    ExecutedQty     DOUBLE(20,8) NOT NULL DEFAULT 0,
    Status     	    varchar(32) NOT NULL,
    TimeInForce     varchar(8) NOT NULL,
    Type     	    varchar(8) NOT NULL,
    Side     	    varchar(8) NOT NULL,
    StopPrice       DOUBLE(20,8) NOT NULL DEFAULT 0,
    IcebergQty      DOUBLE(20,8) NOT NULL DEFAULT 0,
    Time            datetime DEFAULT NULL,
    IsWorking	    tinyint NOT NULL DEFAULT 1,
    LastQueryTime   datetime DEFAULT NULL,
    UNIQUE (ClientOrderID),
    UNIQUE (ProjectID)
  ) comment 'order_list table';
