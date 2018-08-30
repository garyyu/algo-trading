# Algorithm Trading with Binance API

Algorithmic trading (automated trading, black-box trading or simply algo-trading) is the process of using computers programed to follow a defined set of instructions (an algorithm) for placing a trade in order to generate profits at a speed and frequency that is impossible for a human trader. The defined sets of rules are based on timing, price, quantity or any mathematical model. Apart from profit opportunities for the trader, algo-trading makes markets more liquid and makes trading more systematic by ruling out the impact of human emotions on trading activities. 

This project is only for cryptocurrency trading for example BTC, ETH, BNB and so on. For the moment, it's using Binance API to do this, and plan to support other exchanges also.

## Getting started

- Get the code and build
```shell
git clone https://github.com/garyyu/algo-trading.git
cd algo-trading
go build
```

- Before running

1. Some environment variables have to be set before running this `algo-trading`

Propose to create a file named as `binance.env` and write these privacy info in it. But remember to **keep this file safe**!

```shell

# APIKEY
export BINANCE_APIKEY="Iuw9nD****"
export BINANCE_SECRET="Xe3yzK****"

# MySQL
export BINANCE_DB_USER="your_db_user_name"
export BINANCE_DB_PWD="your_db_password"
```
You can find your binance APIKEY and SECRET in your Binance website.

2. MySQL database preparation

Please read some MySQL basic usage document firstly if you're new to database.

Create a new database with name as `binance`.

After your MySQL service start up and `binance` database created, you can set up the database tables for the 1st time:
```shell
mysql -uYour_DB_Username -pYour_DB_Password -Dbinance < db/*.sql 
```
This will set up all the DB tables. 

3. Ready
After setting up all above well, and starting your MySQL database service, then you can run `algo-trading` now:
```shell
souce ~/binance.env
./algo-trading
```

For the first time of running, it could need some time to download some history K-Lines data from Binance API server.


