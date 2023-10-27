package tokens

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestTxErr(t *testing.T) {
	c, err := NewTxErrorRecordContract("https://json-rpc.evm.shimmer.network/", "wss://ws.json-rpc.evm.shimmer.network/", "0xD9B13709Ce4Ef82402c091f3fc8A93a9360A5c1e", 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	orderC := make(chan *TxErrorRecord)
	go c.StartListen(orderC)

	for order := range orderC {
		if order.Error != nil {
			fmt.Println(err)
			if order.Type == 0 {
				t.Fatal(err)
			}
		} else {
			fmt.Println(hex.EncodeToString(order.Txid), order.FromCoin, order.ToCoin)
		}
	}
}
