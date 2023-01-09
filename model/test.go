package model

func AddCollectPendingOrder(account, from, coin, amount string) error {
	_, err := db.Exec("insert into collect_order_pending(`account`,`from`,`coin`,`amount`) values(?,?,?,?)", account, from, coin, amount)
	return err
}

func SetTestPairPool() error {
	return nil
}

func AddLiquidityPendingOrder(account, coin1, coin2, amount1 string) error {
	_, err := db.Exec("insert into liquidity_add_order_pending(`account`,`coin1`,`coin2`,`amount1`) values(?,?,?,?)", account, coin1, coin2, amount1)
	return err
}

func AddSwapPendingOrder(from, fromCoin, fromAmount, to, toCoin, minAmount string) error {
	_, err := db.Exec("insert into swap_order_pending(`from_address`,`from_coin`,`from_amount`,`to_address`,`to_coin`,`min_amount`) values(?,?,?,?,?,?)", from, fromCoin, fromAmount, to, toCoin, minAmount)
	return err
}
