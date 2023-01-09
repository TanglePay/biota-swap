package model

import (
	"fmt"
	"time"
)

func CancelTimeOutOrders() error {
	ts := time.Now().AddDate(0, 0, -1).UnixMilli()
	_, err1 := db.Exec("delete from `liquidity_add_order_pending` where `ts`<?", ts)
	_, err2 := db.Exec("delete from `swap_order_pending` where `ts`<?", ts)
	_, err3 := db.Exec("delete from `collect_order_pending` where `ts`<?", ts)
	if err1 != nil || err2 != nil || err3 != nil {
		return fmt.Errorf("delete timeout pending orders error. %v, %v, %v", err1, err2, err3)
	}
	return nil
}
