package model

import (
	"database/sql"
	"fmt"
	"time"
)

const OverTime = 300

func UpdateAccountState(account string) error {
	_, err := db.Exec("replace into `service`(`account`,`ts`) values(?,?)", account, time.Now().Unix())
	return err
}

func CheckAccountsState(accounts []string) (bool, error) {
	if len(accounts) == 0 {
		return true, nil
	}
	as := "'" + accounts[0] + "'"
	for i := len(accounts) - 1; i > 0; i-- {
		as += ",'" + accounts[i] + "'"
	}
	query := fmt.Sprintf("select `account` from `service` where `account` in (%s) and `ts`<?", as)
	row := db.QueryRow(query, time.Now().Unix()-OverTime)
	var a string
	if err := row.Scan(&a); err == sql.ErrNoRows {
		return true, nil
	} else {
		return false, fmt.Errorf("%s was closed. %v", a, err)
	}
}
