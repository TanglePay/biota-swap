package iotasmr

import (
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/hive.go/serializer/v2"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/builder"
)

type WrapOrder struct {
	Tag    string
	From   string `json:"from"`
	To     string `json:"to"`
	Symbol string `json:"symbol"`
}

// NewUnsignTransactionBuilder creates a new UnsignTransactionBuilder.
func NewUnsignTransactionBuilder(networkID iotago.NetworkID) *UnsignTransactionBuilder {
	return &UnsignTransactionBuilder{
		essence: &iotago.TransactionEssence{
			NetworkID: networkID,
			Inputs:    iotago.Inputs{},
			Outputs:   iotago.Outputs{},
			Payload:   nil,
		},
		inputs: iotago.OutputSet{},
	}
}

// UnsignTransactionBuilder is used to easily build up a Transaction.
type UnsignTransactionBuilder struct {
	essence *iotago.TransactionEssence
	inputs  iotago.OutputSet
}

// AddInput adds the given input to the builder.
func (b *UnsignTransactionBuilder) AddInput(input *builder.TxInput) *UnsignTransactionBuilder {
	b.essence.Inputs = append(b.essence.Inputs, input.InputID.UTXOInput())
	b.inputs[input.InputID] = input.Input
	return b
}

// AddOutput adds the given output to the builder.
func (b *UnsignTransactionBuilder) AddOutput(output iotago.Output) *UnsignTransactionBuilder {
	b.essence.Outputs = append(b.essence.Outputs, output)
	return b
}

// GetTxEssenceDataHash gets the tx essence data for signing
// return hash, *iotago.TransactionEssence, error
func (b *UnsignTransactionBuilder) GetTxEssenceData() ([]byte, []byte, error) {
	var inputIDs iotago.OutputIDs
	for _, input := range b.essence.Inputs {
		inputIDs = append(inputIDs, input.(*iotago.UTXOInput).ID())
	}

	inputs := inputIDs.OrderedSet(b.inputs)
	commitment, err := inputs.Commitment()
	if err != nil {
		return nil, nil, err
	}
	copy(b.essence.InputsCommitment[:], commitment)
	hash, err := b.essence.SigningMessage()
	if err != nil {
		return nil, nil, err
	}
	txData, err := b.essence.MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	return hash, txData, nil
}

// NewBlockBuilder builds the transaction with signature and then swaps to a BlockBuilder with
// the transaction set as its payload.
func NewBlockBuilder(protoParas *iotago.ProtocolParameters, txData []byte, signature iotago.Signature) (*builder.BlockBuilder, error) {
	txEssence := &iotago.TransactionEssence{}
	if err := txEssence.UnmarshalJSON(txData); err != nil {
		return nil, fmt.Errorf("txEssence.UnmarshalJSON error. %s, %v", hex.EncodeToString(txData), err)
	}
	unlocks := iotago.Unlocks{}
	for i := range txEssence.Inputs {
		if i == 0 {
			unlocks = append(unlocks, &iotago.SignatureUnlock{Signature: signature})
		} else {
			unlocks = append(unlocks, &iotago.ReferenceUnlock{Reference: 0})
		}
	}
	sigTxPayload := &iotago.Transaction{Essence: txEssence, Unlocks: unlocks}
	if _, err := sigTxPayload.Serialize(serializer.DeSeriModePerformValidation, protoParas); err != nil {
		return nil, err
	}
	blockBuilder := builder.NewBlockBuilder()
	return blockBuilder.ProtocolVersion(protoParas.Version).Payload(sigTxPayload), nil
}
