drop table if exists algo_demo;


create table algo_demo (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Hours	    int(12) NOT NULL DEFAULT 0,
    StartTime       datetime NOT NULL 
  ) comment 'algo_demo table';
