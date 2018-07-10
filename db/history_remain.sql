drop table if exists history_remain;

create table history_remain (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Amount    	    DOUBLE(20,8) NOT NULL DEFAULT 0,
    Time            datetime DEFAULT NULL,
    UNIQUE (Symbol)
  ) comment 'history_remain table';
