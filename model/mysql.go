package model

import (
	"bwrap/config"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func ConnectToMysql() {
	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Db.Usr, config.Db.Pwd, config.Db.Host, config.Db.Port, config.Db.DbName))
	if err != nil {
		log.Panic(err)
	}

	if err = db.Ping(); nil != err {
		log.Panic("Connect to Mysql error : " + err.Error())
	}
}

func Ping() error {
	if db == nil {
		return fmt.Errorf("mysql connection is nil")
	}
	if err := db.Ping(); nil != err {
		return fmt.Errorf("connect to Mysql error : %v", err)
	}
	return nil
}
