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
}

func StoreSwapOrder(wo *SwapOrder) error {
	_, err := db.Exec("insert into `swap_order`(`txid`,`src_token`,`dest_token`,`wrap`,`from`,`to`,`amount`,`ts`) values(?,?,?,?,?,?,?,?)", wo.TxID, wo.SrcToken, wo.DestToken, wo.Wrap, wo.From, wo.To, wo.Amount, wo.Ts)
	return err
}

func UpdateChainRecord(chainid, txid string, state int) error {
	_, err := db.Exec("update `chain_record` set `state`=? where `chainid`=? and `tx`=?", state, chainid, txid)
	return err
}
