drop table if exists hunt_list;


create table hunt_list (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    ForceEnter	    tinyint NOT NULL DEFAULT 0, 
    Amount    	    DOUBLE(3,2) NOT NULL DEFAULT 0,
    Time    	    datetime NOT NULL 
  ) comment 'hunt_list table';
