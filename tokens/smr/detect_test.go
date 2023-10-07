package smr

import (
	"bwrap/tokens"
	"fmt"
	"testing"
)

func TestListShimmer(t *testing.T) {
	token := NewShimmerToken("https://api.shimmer.network", "0x3983c06c992a7e798c5774c90b0e9fc1fe7d631b707c050fb5375fee6d0d86f3", "SOON", "0884298fe9b82504d26ddb873dbd234a344c120da3a4317d8063dbcf96d356aa9d0100000000", "smr")
	orderC := make(chan *tokens.SwapOrder, 10000)
	go token.StartWrapListen(orderC)
	fmt.Println("Start to listen wrap order")
	for {
		select {
		case order := <-orderC:
			if order.Error != nil {
				fmt.Println(order.Error.Error())
				if order.Type == 0 {
					return
				}
			} else {
				fmt.Printf("Wrap Order : %v\n", *order)
			}
		}
	}
}
