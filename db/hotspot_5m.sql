drop table if exists hotspot_5m;


create table hotspot_5m (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    HotRank	    int(12) NOT NULL DEFAULT 0,
    HighLowRatio    DOUBLE(20,8) NOT NULL DEFAULT 0,
    VolumeRatio     DOUBLE(20,8) NOT NULL DEFAULT 0,
    CloseOpenRatio  DOUBLE(20,8) NOT NULL DEFAULT 0,
    HLRxVR    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Time    	    datetime NOT NULL 
  ) comment 'hotspot_5m table';
