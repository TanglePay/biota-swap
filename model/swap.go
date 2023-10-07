package model

type SwapOrder struct {
	ID        int64
	TxID      string
	SrcToken  string
	DestToken string
	Wrap      int
	From      string
	To        string
	Amount    string
	Hash      string
	State     int
	Ts        int64
	Org       string
}

func StoreSwapOrder(wo *SwapOrder) error {
	if len(wo.Org) == 0 {
		wo.Org = "IotaBee"
	}
	_, err := db.Exec("insert into `swap_order`(`txid`,`src_token`,`dest_token`,`wrap`,`from`,`to`,`amount`,`ts`,`org`) values(?,?,?,?,?,?,?,?,?)", wo.TxID, wo.SrcToken, wo.DestToken, wo.Wrap, wo.From, wo.To, wo.Amount, wo.Ts, wo.Org)
	return err
}

func MoveOrderToError(wo *SwapOrder) error {
	tx, err := db.Begin()
	if err != nil {
		if tx != nil {
			tx.Rollback()
		}
		return err
	}

	if _, err = tx.Exec("delete from `swap_order` where `txid`=?", wo.TxID); err != nil {
		tx.Rollback()
		return err
	}

	if _, err = tx.Exec("insert into `error_order`(`txid`,`src_token`,`dest_token`,`wrap`,`from`,`to`,`amount`,`hash`,`state`,`ts`,`org`) values(?,?,?,?,?,?,?,?,?,?,?)", wo.TxID, wo.SrcToken, wo.DestToken, wo.Wrap, wo.From, wo.To, wo.Amount, wo.Hash, wo.State, wo.Ts, wo.Org); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func UpdateChainRecord(wo *SwapOrder) error {
	_, err := db.Exec("update `swap_order` set `hash`=?, `state`=? where txid=?", wo.Hash, wo.State, wo.TxID)
	return err
}
