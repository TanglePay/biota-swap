package model

import (
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

func addVolumeToSwapPair(tx *sql.Tx, pair string, amount1, amount2 *big.Int) error {
	id := time.Now().Unix() / 86400
	row := tx.QueryRow("select `amount1`,`amount2` from `volume` where `id`=? and `pair`=? for update", id, pair)
	var str1, str2 string
	if err := row.Scan(&str1, &str2); err != nil {
		if err != sql.ErrNoRows {
			return fmt.Errorf("Get volume error. %v", err)
		} else {
			str1, str2 = "0", "0"
		}
	}

	a1, b1 := new(big.Int).SetString(str1, 10)
	a2, b2 := new(big.Int).SetString(str2, 10)
	if !b1 || !b2 {
		return fmt.Errorf("Convert amount1 and amount2 to big.Int error. %s,%s", str1, str2)
	}

	a1.Add(a1, amount1)
	a2.Add(a2, amount2)

	_, err := tx.Exec("replace into `volume`(`id`,`pair`,`amount1`,`amount2`) values(?,?,?,?)", id, pair, a1.String(), a2.String())
	return err
}
