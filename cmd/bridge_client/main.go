package main

import "fmt"

func main() {
	fmt.Println("1. Wrap")
	fmt.Println("2. UnWrap")
	fmt.Println("3. Submit Signer Proposal")
	fmt.Println("4. Agree Signer Proposal")
	fmt.Println("5. Submit RequireCount Proposal")
	fmt.Println("6. Agree RequireCount Proposal")
	fmt.Println("7. Withdraw Fee")
	fmt.Println("8. UnWrap Fee")
	for {
		fmt.Printf("Test Bridge, choose an item to run (0 to quit) : ")
		item := 0
		fmt.Scanf("%d", &item)
		switch item {
		case 1:
			//WrapIota()
		case 2:
			//WrapETH()
		case 3:
			//WrapErc20()
		case 4:
			//UnwrapIota()
		case 5:
			//UnwrapETH()
		case 6:
			//UnwrapWBTC()
		case 0:
			return
		}
	}
}
