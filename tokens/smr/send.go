package smr

import (
	"bwrap/gl"
	"bwrap/tokens"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/builder"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func (st *ShimmerToken) MultiSignType() int {
	return tokens.CenterSign
}

func (st *ShimmerToken) Symbol() string {
	return st.symbol
}

func (st *ShimmerToken) PublicKey() []byte {
	return st.publicKey
}

func (st *ShimmerToken) KeyType() string {
	return "EC256K1"
}

func (st *ShimmerToken) Address() string {
	return st.walletAddr
}

func (st *ShimmerToken) CheckSentTx(txid []byte) (bool, error) {
	return true, nil
}

func (st *ShimmerToken) CheckUserTx(txid []byte, toCoin string, d int) (string, string, *big.Int, error) {
	return "", "", nil, nil
}

func (st *ShimmerToken) CheckTxFailed(failedTx, txid []byte, to string, amount *big.Int, d int) error {
	return nil
}

func (st *ShimmerToken) CheckUnWrapTx(txid []byte, to, symbol string, amount *big.Int) error {
	return nil
}

func (st *ShimmerToken) SendSignedTxData(signedHash string, txData []byte) ([]byte, error) {
	return nil, nil
}

func (st *ShimmerToken) SendWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	return nil, nil
}

func (st *ShimmerToken) SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	toAddr, err := iotago.ParseEd25519AddressFromHexString("0x" + to)
	if err != nil {
		return nil, fmt.Errorf("send shimmer address error. %s, %v", to, err)
	}

	addr, signer, err := st.getWalletAddress(prv)
	if err != nil {
		return nil, err
	}

	info, err := st.nodeAPI.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get shimmerr node info error. %v", err)
	}

	txBuilder := builder.NewTransactionBuilder(info.Protocol.NetworkID())
	outputTo := &iotago.BasicOutput{
		NativeTokens: iotago.NativeTokens{
			&iotago.NativeToken{
				ID:     st.tokenID,
				Amount: amount,
			},
		},
		Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
			Address: toAddr,
		}},
	}
	outputTo.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(outputTo.VBytes(&info.Protocol.RentStructure, nil))
	txBuilder.AddOutput(outputTo)
	leftTokenAmount, leftSmrAmount, err := st.getNativeTokenOutputs(txBuilder, amount, st.hrp, addr)
	if err != nil {
		return nil, fmt.Errorf("get native token outputs error. %s,%s, %v", st.tokenID.ToHex(), addr.Bech32(st.hrp), err)
	}
	needSmrAmount := outputTo.Amount
	if leftTokenAmount.Cmp(big.NewInt(0)) > 0 {
		outputSelf := &iotago.BasicOutput{
			NativeTokens: iotago.NativeTokens{
				&iotago.NativeToken{
					ID:     st.tokenID,
					Amount: leftTokenAmount,
				},
			},
			Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
				Address: addr,
			}},
		}
		outputSelf.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(outputSelf.VBytes(&info.Protocol.RentStructure, nil))
		needSmrAmount += outputSelf.Amount
		txBuilder.AddOutput(outputSelf)
	}
	if needSmrAmount != leftSmrAmount {
		left, err := st.getBasiceUnSpentOutputs(txBuilder, st.hrp, addr)
		if err != nil {
			return nil, fmt.Errorf("get basic shimmer outputs error. %s, %v", addr.Bech32(st.hrp), err)
		}
		left += leftSmrAmount
		smrOutput := &iotago.BasicOutput{
			Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
				Address: addr,
			}},
		}
		smrOutput.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(smrOutput.VBytes(&info.Protocol.RentStructure, nil))
		if left < (needSmrAmount + smrOutput.Amount) {
			return nil, fmt.Errorf("balance amount is not enough. %d : %d", needSmrAmount+smrOutput.Amount, left)
		}
		smrOutput.Amount = left - needSmrAmount
		txBuilder.AddOutput(smrOutput)
	}

	txBuilder.AddTaggedDataPayload(&iotago.TaggedData{Tag: []byte("TpBridge")})

	blockBuilder := txBuilder.BuildAndSwapToBlockBuilder(&info.Protocol, signer, nil)

	block, err := blockBuilder.Tips(context.Background(), st.nodeAPI).
		ProofOfWork(context.Background(), &info.Protocol, float64(info.Protocol.MinPoWScore)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build block error. %v", err)
	}
	id, err := st.nodeAPI.SubmitBlock(context.Background(), block, &info.Protocol)
	if err != nil {
		return nil, fmt.Errorf("Send block to node error. %v", err)
	}

	return id[:], nil
}

func (st *ShimmerToken) CreateUnWrapTxData(addr string, amount *big.Int, extra []byte) ([]byte, []byte, error) {
	return nil, nil, fmt.Errorf("Don't support this method")
}

func (st *ShimmerToken) ValiditeUnWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	return tokens.BaseTransaction{}, fmt.Errorf("Don't support this method")
}

func (st *ShimmerToken) getWalletAddress(prv *ecdsa.PrivateKey) (iotago.Address, iotago.AddressSigner, error) {
	pk := crypto.FromECDSA(prv)
	pk = append(pk, st.publicKey...)
	addr := iotago.Ed25519AddressFromPubKey(st.publicKey)
	addrKeys := iotago.NewAddressKeysForEd25519Address(&addr, pk)
	signer := iotago.NewInMemoryAddressSigner(addrKeys)
	return &addr, signer, nil
}

func (st *ShimmerToken) getNativeTokenOutputs(b *builder.TransactionBuilder, amount *big.Int, prefix iotago.NetworkPrefix, addr iotago.Address) (*big.Int, uint64, error) {
	indexer, err := st.nodeAPI.Indexer(context.Background())
	if err != nil {
		return nil, 0, err
	}

	hasNativeToken := true
	query := nodeclient.BasicOutputsQuery{
		AddressBech32: addr.Bech32(prefix),
		IndexerNativeTokenParas: nodeclient.IndexerNativeTokenParas{
			HasNativeTokens: &hasNativeToken,
		},
	}
	res, err := indexer.Outputs(context.Background(), &query)
	if err != nil {
		return nil, 0, err
	}
	sum := big.NewInt(0)
	sumSmr := uint64(0)
	count := 0
	for res.Next() {
		ids, err := res.Response.Items.OutputIDs()
		if err != nil {
			return nil, 0, err
		}
		outputs, _ := res.Outputs()
		for i, output := range outputs {
			if len(output.NativeTokenList()) != 1 {
				continue
			}
			token := output.NativeTokenList()[0]
			if token.ID != st.tokenID {
				continue
			}
			sum.Add(sum, token.Amount)
			sumSmr += output.Deposit()
			b.AddInput(&builder.TxInput{UnlockTarget: addr, Input: output, InputID: ids[i]})
			count++
			if count >= gl.MAX_INPUT_COUNT/2 {
				break
			}
		}
	}
	if sum.Cmp(amount) < 0 {
		return amount, sumSmr, fmt.Errorf("balance amount is not enough. %s : %s", sum.String(), amount.String())
	}
	return new(big.Int).Sub(sum, amount), sumSmr, nil
}

func (st *ShimmerToken) getBasiceUnSpentOutputs(b *builder.TransactionBuilder, prefix iotago.NetworkPrefix, addr iotago.Address) (uint64, error) {
	indexer, err := st.nodeAPI.Indexer(context.Background())
	if err != nil {
		return 0, err
	}

	hasNativeToken := false
	query := nodeclient.BasicOutputsQuery{
		AddressBech32: addr.Bech32(prefix),
		IndexerNativeTokenParas: nodeclient.IndexerNativeTokenParas{
			HasNativeTokens: &hasNativeToken,
		},
	}
	res, err := indexer.Outputs(context.Background(), &query)
	if err != nil {
		return 0, err
	}
	sum := uint64(0)
	count := 0
	for res.Next() {
		ids, err := res.Response.Items.OutputIDs()
		if err != nil {
			return 0, err
		}
		outputs, _ := res.Outputs()
		for i, output := range outputs {
			if len(output.NativeTokenList()) > 0 {
				continue
			}
			b.AddInput(&builder.TxInput{UnlockTarget: addr, Input: output, InputID: ids[i]})
			sum += output.Deposit()
			count++
			if count >= gl.MAX_INPUT_COUNT/2 {
				break
			}
		}
	}
	return sum, nil
}
