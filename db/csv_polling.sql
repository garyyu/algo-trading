drop table if exists csv_polling;


create table csv_polling (
    id int primary key auto_increment,
    Symbol	    varchar(16) NOT NULL,
    Type  	    ENUM('FiveMinutes', 'Hour', 'Day') NOT NULL DEFAULT 'FiveMinutes',
    StartTime       datetime NOT NULL 
  ) comment 'csv_polling table';
