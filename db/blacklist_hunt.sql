drop table if exists blacklist_hunt;


create table blacklist_hunt (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Reason	    varchar(256) NOT NULL,
    Time    	    datetime NOT NULL 
  ) comment 'blacklist_hunt table';
