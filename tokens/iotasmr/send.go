package iotasmr

import (
	"bwrap/gl"
	"bwrap/tokens"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/builder"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func (ist *IotaSmrToken) MultiSignType() int {
	return tokens.SmpcSign
}

func (ist *IotaSmrToken) ChainID() string {
	return string(ist.hrp)
}

func (ist *IotaSmrToken) Symbol() string {
	return strings.ToUpper(ist.symbol)
}

func (ist *IotaSmrToken) PublicKey() []byte {
	return ist.publicKey
}

func (ist *IotaSmrToken) KeyType() string {
	return "ED25519"
}

func (ist *IotaSmrToken) Address() string {
	return ist.addrBech32
}

// check the unwrap order tx state which was sent.
func (ist *IotaSmrToken) CheckSentTx(txid []byte) (bool, error) {
	bid := iotago.EmptyBlockID()
	if len(txid) != 32 {
		return true, fmt.Errorf("txid error. %s", hex.EncodeToString(txid))
	}
	copy(bid[:], txid)

	res, err := ist.nodeAPI.BlockMetadataByBlockID(context.Background(), bid)
	if err != nil {
		return true, err
	}
	if !res.Solid {
		return true, fmt.Errorf("txid has not solid. %s", hex.EncodeToString(txid))
	}
	if res.ConflictReason != 0 {
		return false, fmt.Errorf("%d:%s", res.ConflictReason, res.LedgerInclusionState)
	}
	return true, nil
}

// check user's wrap order, using the blockID
func (ist *IotaSmrToken) CheckUserTx(txid []byte, toCoin string, d int) (string, string, *big.Int, error) {
	if d != 1 {
		return "", "", nil, fmt.Errorf("iota network d error. %d", d)
	}
	bid := iotago.EmptyBlockID()
	if len(txid) != 32 {
		return "", "", nil, fmt.Errorf("txid error. %s", hex.EncodeToString(txid))
	}
	copy(bid[:], txid)

	meta, err := ist.nodeAPI.BlockMetadataByBlockID(context.Background(), bid)
	if bid.Empty() || err != nil {
		return "", "", nil, fmt.Errorf("BlockMetadataByBlockID error. %s, %v", bid.String(), err)
	}
	if meta.ConflictReason != 0 {
		return "", "", nil, fmt.Errorf("ConflictReason is not confirm. %s : %d", bid.String(), meta.ConflictReason)
	}

	block, err := ist.nodeAPI.BlockByBlockID(context.Background(), bid, ist.protoParas)
	if err != nil {
		return "", "", nil, fmt.Errorf("BlockByBlockID error. %v", err)
	}

	if block.Payload == nil || block.Payload.PayloadType() != iotago.PayloadTransaction {
		return "", "", nil, fmt.Errorf("payload type error. %s", bid.String())
	}

	outputs, err := block.Payload.(*iotago.Transaction).OutputsSet()
	if err != nil {
		return "", "", nil, fmt.Errorf("tx.OutputsSet() error. %s, %v", bid.String(), err)
	}

	var order *tokens.SwapOrder = nil
	for _, output := range outputs {
		unlock := output.UnlockConditionSet().Address()
		if unlock != nil && unlock.Address.Equal(ist.addr) {
			order, err = ist.getWrapOrderByOutput(output, meta.BlockID)
			break
		}
	}
	if order == nil {
		return "", "", nil, fmt.Errorf("don't found wrap order error. %s, %v", bid.String(), err)
	}

	return order.From, order.To, order.Amount, nil
}

func (ist *IotaSmrToken) CheckTxFailed(failedTx, txid []byte, to string, amount *big.Int, d int) error {
	if d != -1 {
		return fmt.Errorf("iota network d error. %d", d)
	}

	var blockID iotago.BlockID
	copy(blockID[:], failedTx)
	meta, err := ist.nodeAPI.BlockMetadataByBlockID(context.Background(), blockID)
	if err != nil {
		return fmt.Errorf("MessageMetadataByMessageID error. %s, %v", blockID.String(), err)
	}
	if meta.ConflictReason == 0 {
		return fmt.Errorf("tx success. %s", blockID.String())
	}

	block, err := ist.nodeAPI.BlockByBlockID(context.Background(), blockID, ist.protoParas)
	if err != nil {
		return fmt.Errorf("MessageByMessageID error. %s, %v", blockID.String(), err)
	}

	if block.Payload == nil || block.Payload.PayloadType() != iotago.PayloadTransaction {
		return fmt.Errorf("block payload type error. %s", blockID.String())
	}

	unlocks := block.Payload.(*iotago.Transaction).Unlocks
	if len(unlocks) < 1 || unlocks[0].Type() != iotago.UnlockSignature {
		return fmt.Errorf("block unlocks type error. %s", blockID.String())
	}
	unlock := unlocks[0].(*iotago.SignatureUnlock).Signature
	if unlock.Type() != iotago.SignatureEd25519 {
		return fmt.Errorf("block unlocks type error. %s", blockID.String())
	}
	if !bytes.Equal(unlock.(*iotago.Ed25519Signature).PublicKey[:], ist.publicKey) {
		return fmt.Errorf("publickeys are not equal. %s", hex.EncodeToString(unlock.(*iotago.Ed25519Signature).PublicKey[:]))
	}

	outputs, err := block.Payload.(*iotago.Transaction).OutputsSet()
	for _, output := range outputs {
		unlock := output.UnlockConditionSet().Address()
		if unlock != nil {
			continue
		}
		if unlock.Address.Bech32(ist.hrp) == to {
			a := ist.getAmountFromOutput(output)
			if a.Cmp(amount) != 0 {
				return fmt.Errorf("amounts are not equal. %s : %s", a.String(), amount.String())
			}
			tag := ist.getTagDataFromFeature(output.FeatureSet())
			if !bytes.Equal(txid, tag) {
				return fmt.Errorf("txids are not equal. %s : %s", hex.EncodeToString(tag), hex.EncodeToString(txid))
			}
			return nil
		}
	}
	return fmt.Errorf("can't find the to addess. %s : %s : %v", blockID.String(), to, err)
}

func (ist *IotaSmrToken) CreateUnWrapTxData(ed25519Id string, amount *big.Int, tag []byte) ([]byte, []byte, error) {
	addrBytes := common.FromHex(ed25519Id)
	if len(addrBytes) != 32 {
		return nil, nil, fmt.Errorf("toAddress error. %s", ed25519Id)
	}
	addr := &iotago.Ed25519Address{}
	copy(addr[:], addrBytes)
	if ist.tokenID.Empty() {
		return ist.getUnSignedTxDataBasic(addr, amount, tag)
	}
	return ist.getUnSignedTxDataNativeToken(addr, amount, tag)
}

func (ist *IotaSmrToken) SendSignedTxData(signedHash string, txData []byte) ([]byte, error) {
	ed25519Sig := &iotago.Ed25519Signature{}
	copy(ed25519Sig.Signature[:], signedHash)
	copy(ed25519Sig.PublicKey[:], ist.publicKey)

	blockBuilder, err := NewBlockBuilder(ist.protoParas, txData, ed25519Sig)
	if err != nil {
		return nil, err
	}

	block, err := blockBuilder.Tips(context.Background(), ist.nodeAPI).
		ProofOfWork(context.Background(), ist.protoParas, float64(ist.protoParas.MinPoWScore)).
		Build()
	if err != nil {
		return nil, err
	}
	id, err := ist.nodeAPI.SubmitBlock(context.Background(), block, ist.protoParas)
	if err != nil {
		return nil, err
	}
	return id[:], err
}

func (ist *IotaSmrToken) SendWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	return nil, nil
}

func (ist *IotaSmrToken) SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	return nil, fmt.Errorf("don't support this method")
}

func (ist *IotaSmrToken) ValiditeUnWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	baseTx := tokens.BaseTransaction{}

	seri := &iotago.TransactionEssence{}
	if err := seri.UnmarshalJSON(txData); err != nil {
		return baseTx, fmt.Errorf("msgContext can't be UnmarshalJSON to TransactionEssence. %v", err)
	}

	if sign, err := seri.SigningMessage(); err != nil {
		return baseTx, fmt.Errorf("seri.SigningMessage error. %v", err)
	} else if !bytes.Equal(hash, sign) {
		return baseTx, fmt.Errorf("hash is not right. %s : %s", hex.EncodeToString(hash), hex.EncodeToString(sign))
	}

	if len(seri.Outputs) < 1 || seri.Outputs[0].Type() != iotago.OutputBasic {
		return baseTx, fmt.Errorf("tx output error")
	}
	output := seri.Outputs[0].(*iotago.BasicOutput)

	if output.UnlockConditionSet().Address() == nil {
		return baseTx, fmt.Errorf("output unlock type error")
	}
	baseTx.To = output.UnlockConditionSet().Address().Address.String()
	baseTx.Amount = ist.getAmountFromOutput(output)
	baseTx.Txid = ist.getTagDataFromFeature(output.FeatureSet())
	return baseTx, nil
}

func (ist *IotaSmrToken) getUnSignedTxDataBasic(toAddr iotago.Address, amount *big.Int, tag []byte) ([]byte, []byte, error) {
	sendAmount := amount.Uint64()

	info, err := ist.nodeAPI.Info(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("get iotasmr node info error. %v", err)
	}

	txBuilder := NewUnsignTransactionBuilder(info.Protocol.NetworkID())

	output := iotago.BasicOutput{
		Amount: sendAmount,
		Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
			Address: toAddr,
		}},
		Features: iotago.Features{
			&iotago.TagFeature{
				Tag: tag,
			},
			&iotago.MetadataFeature{
				Data: []byte("Unwrap"),
			},
		},
	}
	left, err := ist.getBasiceUnSpentOutputs(txBuilder, sendAmount)
	if err != nil {
		return nil, nil, err
	}
	txBuilder.AddOutput(&output)
	if left > 0 {
		txBuilder.AddOutput(&iotago.BasicOutput{
			Amount: left,
			Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
				Address: ist.addr,
			}},
		})
	}

	return txBuilder.GetTxEssenceData()
}

func (ist *IotaSmrToken) getUnSignedTxDataNativeToken(toAddr iotago.Address, amount *big.Int, tag []byte) ([]byte, []byte, error) {
	info, err := ist.nodeAPI.Info(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("get iotasmr node info error. %v", err)
	}

	txBuilder := NewUnsignTransactionBuilder(info.Protocol.NetworkID())
	outputTo := &iotago.BasicOutput{
		NativeTokens: iotago.NativeTokens{
			&iotago.NativeToken{
				ID:     ist.tokenID,
				Amount: amount,
			},
		},
		Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
			Address: toAddr,
		}},
		Features: iotago.Features{
			&iotago.TagFeature{
				Tag: tag,
			},
			&iotago.MetadataFeature{
				Data: []byte("Unwrap"),
			},
		},
	}
	outputTo.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(outputTo.VBytes(&info.Protocol.RentStructure, nil))
	txBuilder.AddOutput(outputTo)
	leftTokenAmount, leftSmrAmount, err := ist.getNativeTokenOutputs(txBuilder, amount)
	if err != nil {
		return nil, nil, fmt.Errorf("get native token outputs error. %s,%s, %v", ist.tokenID.String(), ist.addrBech32, err)
	}
	needSmrAmount := outputTo.Amount
	if leftTokenAmount.Cmp(big.NewInt(0)) > 0 {
		outputSelf := &iotago.BasicOutput{
			NativeTokens: iotago.NativeTokens{
				&iotago.NativeToken{
					ID:     ist.tokenID,
					Amount: leftTokenAmount,
				},
			},
			Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
				Address: ist.addr,
			}},
		}
		outputSelf.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(outputSelf.VBytes(&info.Protocol.RentStructure, nil))
		needSmrAmount += outputSelf.Amount
		txBuilder.AddOutput(outputSelf)
	}
	if needSmrAmount != leftSmrAmount {
		left, err := ist.getBasiceUnSpentOutputs(txBuilder, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("get basic shimmer outputs error. %s, %v", ist.addrBech32, err)
		}
		left += leftSmrAmount
		smrOutput := &iotago.BasicOutput{
			Conditions: iotago.UnlockConditions{&iotago.AddressUnlockCondition{
				Address: ist.addr,
			}},
		}
		smrOutput.Amount = uint64(info.Protocol.RentStructure.VByteCost) * uint64(smrOutput.VBytes(&info.Protocol.RentStructure, nil))
		if left < (needSmrAmount + smrOutput.Amount) {
			return nil, nil, fmt.Errorf("balance amount is not enough. %d : %d", needSmrAmount+smrOutput.Amount, left)
		}
		smrOutput.Amount = left - needSmrAmount
		txBuilder.AddOutput(smrOutput)
	}
	return txBuilder.GetTxEssenceData()
}

func (ist *IotaSmrToken) getBasiceUnSpentOutputs(b *UnsignTransactionBuilder, amount uint64) (uint64, error) {
	indexer, err := ist.nodeAPI.Indexer(context.Background())
	if err != nil {
		return 0, err
	}

	notHas := false
	query := nodeclient.BasicOutputsQuery{
		AddressBech32: ist.addrBech32,
		IndexerNativeTokenParas: nodeclient.IndexerNativeTokenParas{
			HasNativeTokens: &notHas,
		},
		IndexerTimelockParas: nodeclient.IndexerTimelockParas{
			HasTimelock: &notHas,
		},
		IndexerExpirationParas: nodeclient.IndexerExpirationParas{
			HasExpiration: &notHas,
		},
		IndexerStorageDepositParas: nodeclient.IndexerStorageDepositParas{
			HasStorageDepositReturn: &notHas,
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
			b.AddInput(&builder.TxInput{UnlockTarget: ist.addr, Input: output, InputID: ids[i]})
			sum += output.Deposit()
			count++
			if count >= gl.MAX_INPUT_COUNT {
				break
			}
		}
	}
	if sum < amount {
		return amount, fmt.Errorf("balance amount is not enough")
	}
	return sum - amount, nil
}

func (ist *IotaSmrToken) getNativeTokenOutputs(b *UnsignTransactionBuilder, amount *big.Int) (*big.Int, uint64, error) {
	indexer, err := ist.nodeAPI.Indexer(context.Background())
	if err != nil {
		return nil, 0, err
	}

	has := true
	notHas := false
	query := nodeclient.BasicOutputsQuery{
		AddressBech32: ist.addrBech32,
		IndexerNativeTokenParas: nodeclient.IndexerNativeTokenParas{
			HasNativeTokens: &has,
		},
		IndexerTimelockParas: nodeclient.IndexerTimelockParas{
			HasTimelock: &notHas,
		},
		IndexerExpirationParas: nodeclient.IndexerExpirationParas{
			HasExpiration: &notHas,
		},
		IndexerStorageDepositParas: nodeclient.IndexerStorageDepositParas{
			HasStorageDepositReturn: &notHas,
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
			if token.ID != ist.tokenID {
				continue
			}
			sum.Add(sum, token.Amount)
			sumSmr += output.Deposit()
			b.AddInput(&builder.TxInput{UnlockTarget: ist.addr, Input: output, InputID: ids[i]})
			count++
			if count >= gl.MAX_INPUT_COUNT {
				break
			}
		}
	}
	if sum.Cmp(amount) < 0 {
		return amount, sumSmr, fmt.Errorf("balance amount is not enough. %s : %s", sum.String(), amount.String())
	}
	return new(big.Int).Sub(sum, amount), sumSmr, nil
}
