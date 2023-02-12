package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestWrap(t *testing.T) {
	chainClient, err := NewEvmToken("https://rpc-mumbai.maticvigil.com/", "0x9b5EE326F61cc57A94820C6CA36E7795D3A21281", "", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pk := "39e10402beff72d338b4c16b5094f88c94330aa32a17351bf5be05da92671a4d"
	privateKey, err := crypto.HexToECDSA(pk)

	var txid [32]byte
	txid[0] = 243
	txid[1] = 235
	to := common.HexToAddress("0x45ae5c97D8e6598a693F6859847ca1e93b63d14e")
	h, txData, _ := chainClient.CreateWrapTxData(to.Hex(), big.NewInt(100000000), hex.EncodeToString(txid[:]))

	signHash, _ := crypto.Sign(h[:], privateKey)

	rawTx := &types.Transaction{}
	rawTx.UnmarshalJSON(txData)

	signedTx, _ := rawTx.WithSignature(types.NewEIP155Signer(chainClient.chainId), signHash)

	chainClient.client.SendTransaction(context.Background(), signedTx)

	fmt.Println(signedTx.Hash().Hex())
}
