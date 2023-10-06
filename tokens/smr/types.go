package smr

import iotago "github.com/iotaledger/iota.go/v3"

type BlockPayload struct {
	From    string
	Essence PayloadEssence  `json:"essence"`
	Unlocks []PayloadUnlock `json:"unlocks"`
}

type PayloadEssence struct {
	Type    int `json:"type"`
	Payload struct {
		Type iotago.PayloadType `json:"type"`
		Tag  string             `json:"tag"`
		Data string             `json:"data"`
	} `json:"payload"`
}

type PayloadUnlock struct {
	Type      int `json:"type"`
	Signature struct {
		Type      int    `json:"type"`
		PublicKey string `json:"publicKey"`
		Signature string `json:"signature"`
	} `json:"signature"`
}

type WrapOrder struct {
	Tag    string
	From   string `json:"from"`
	To     string `json:"to"`
	Symbol string `json:"symbol"`
}
