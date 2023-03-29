package tokens

import (
	"fmt"
	"testing"
)

func TestTxErr(t *testing.T) {
	c, err := NewTxErrorRecordContract("rpc-mumbai.maticvigil.com", "0xfb55F7f7694F22658FfE6d0fDE37D39384996C4a", ScanBlock, 10)
	if err != nil {
		t.Fatal(err)
	}

	orderC := make(chan *TxErrorRecord)
	go c.StartListen(orderC)

	for {
		select {
		case order := <-orderC:
			if order.Error != nil {
				fmt.Println(err)
				if order.Type == 0 {
					t.Fatal(err)
				}
			} else {
				fmt.Println(*order)
			}
		}
	}
}
