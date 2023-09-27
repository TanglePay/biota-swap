package smr

import iotago "github.com/iotaledger/iota.go/v3"

type BlockPayload struct {
	id      string
	Type    iotago.PayloadType `json:"type"`
	Essence PayloadEssence     `json:"essence"`
	Unlocks []PayloadUnlock    `json:"unlocks"`
}

type PayloadEssence struct {
	Type             int                    `json:"type"`
	NetworkId        iotago.NetworkID       `json:"networkId,string"`
	Inputs           []PayloadEssenceInput  `json:"inputs"`
	InputsCommitment string                 `json:"inputsCommitment"`
	Outputs          []PayloadEssenceOutput `json:"outputs"`
	Payload          struct {
		Type iotago.PayloadType `json:"type"`
		Tag  string             `json:"tag"`
		Data string             `json:"data"`
	} `json:"payload"`
}

type PayloadEssenceInput struct {
	Type                   int    `json:"type"`
	TransactionId          string `json:"transactionId"`
	TransactionOutputIndex int    `json:"transactionOutputIndex"`
}

type PayloadEssenceOutput struct {
	Type             iotago.OutputType                      `json:"type"`
	Amount           uint64                                 `json:"amount,string"`
	NativeTokens     []PayloadEssenceOutputNativeToken      `json:"nativeTokens"`
	UnlockConditions []PayloadEssenceOutputUnlockConditions `json:"unlockConditions"`
}

type PayloadEssenceOutputNativeToken struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

type PayloadEssenceOutputUnlockConditions struct {
	Type    iotago.UnlockConditionType `json:"type"`
	Address struct {
		Type       int    `json:"type"`
		PubKeyHash string `json:"pubKeyHash"`
	} `json:"address"`
}

type PayloadEssencePayload struct {
	Type             int                                    `json:"type"`
	UnlockConditions []PayloadEssenceOutputUnlockConditions `json:"unlockConditions"`
}

type PayloadUnlock struct {
	Type      int `json:"type"`
	Signature struct {
		Type      int    `json:"type"`
		PublicKey string `json:"publicKey"`
		Signature string `json:"signature"`
	} `json:"signature"`
}

type EssencePayloadData struct {
	To     string `json:"to"`
	Symbol string `json:"symbol"`
}
