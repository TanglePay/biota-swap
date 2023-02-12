package evm

import (
	"bwrap/gl"
	"bwrap/tokens"
	"bwrap/tools/crypto"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var MethodWrap = crypto.Keccak256Hash([]byte("wrap(bytes32,uint256,address)"))
var MethodUnWrap = crypto.Keccak256Hash([]byte("send(bytes32,uint256,address)"))
var EventUnWrap = crypto.Keccak256Hash([]byte("UnWrap(address,bytes32,bytes32,uint256)"))

type EvmToken struct {
	client          *ethclient.Client
	url             string
	chainId         *big.Int
	contract        common.Address
	publicKey       []byte
	address         common.Address
	unwrapNetPrefix string
	unwrapChain     string
	privateKey      *ecdsa.PrivateKey
	ListenType      int //0: listen event, 1: scan block
}

func NewEvmToken(uri string, conAddr, publicKey string, prv *ecdsa.PrivateKey, _listenType int) (*EvmToken, error) {
	c, err := ethclient.Dial("https://" + uri)
	if err != nil {
		return nil, err
	}
	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}
	pk := common.Hex2Bytes(publicKey)
	newPk, err := crypto.UnmarshalPubkey(pk)
	if err != nil {
		return nil, err
	}

	return &EvmToken{
		url:        uri,
		client:     c,
		chainId:    chainId,
		contract:   common.HexToAddress(conAddr),
		publicKey:  pk,
		address:    crypto.PubkeyToAddress(*newPk),
		privateKey: prv,
		ListenType: _listenType,
	}, err
}

func (ei *EvmToken) MultiSignType() int {
	return tokens.Contract
}

func (ei *EvmToken) Symbol() string {
	return "SMIOTA"
}

func (ei *EvmToken) PublicKey() []byte {
	return ei.publicKey
}

func (ei *EvmToken) KeyType() string {
	return "EC256K1"
}

func (ei *EvmToken) Address() string {
	return ei.address.Hex()
}

func (ei *EvmToken) CreateWrapTxData(to string, amount *big.Int, txID string) ([]byte, []byte, error) {
	var data []byte
	data = append(data, MethodWrap[:4]...)
	data = append(data, common.Hex2Bytes(txID)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	value := big.NewInt(0)

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("Get SuggestGasPrice error. %v", err)
	}

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.address)
	if err != nil {
		return nil, nil, err
	}
	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)
	h := types.NewEIP155Signer(ei.chainId).Hash(tx)

	txData, _ := tx.MarshalJSON()
	return h[:], txData, nil
}

func (ei *EvmToken) CheckTxData(txid []byte, to string, amount *big.Int) error {
	hash := common.BytesToHash(txid)
	tx, isPending, err := ei.client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return fmt.Errorf("client.TransactionByHash error. %s, %v", hash.Hex(), err)
	}
	if isPending {
		return fmt.Errorf("tx is pending status. %s", hash.Hex())
	}

	data := tx.Data()
	if bytes.Compare(data[:4], MethodUnWrap[:4]) != 0 {
		return fmt.Errorf("tx is not UnWrap.")
	}
	data = data[4:]

	if bytes.Compare(common.Hex2Bytes(to), data[:32]) != 0 {
		return fmt.Errorf("to address is not equal. %s : %s", to, common.Bytes2Hex(data[:32]))
	}

	a := new(big.Int).SetBytes(data[32:])
	if a.Cmp(amount) != 0 {
		return fmt.Errorf("amount is not equal. %d : %d", amount.Uint64(), a.Uint64())
	}

	return nil
}

func (ei *EvmToken) ValiditeWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	baseTx := tokens.BaseTransaction{}

	rawTx := &types.Transaction{}
	rawTx.UnmarshalJSON(txData)

	data := rawTx.Data()
	if bytes.Compare(data[:4], MethodWrap[:4]) != 0 {
		return baseTx, fmt.Errorf("tx method is not right.")
	}
	data = data[4:]

	baseTx.Txid = data[:32]
	data = data[32:]

	baseTx.To = common.BytesToAddress(data[12:32]).Hex()

	h := types.NewEIP155Signer(ei.chainId).Hash(rawTx)
	if bytes.Compare(hash, h.Bytes()) != 0 {
		return baseTx, fmt.Errorf("hash is not right. %s : %s", h.Hex(), hex.EncodeToString(hash))
	}
	return baseTx, nil
}

func (ei *EvmToken) SendSignedTxData(signedHash string, txData []byte) ([]byte, error) {
	rawTx := &types.Transaction{}
	rawTx.UnmarshalJSON(txData)
	signedTx, _ := rawTx.WithSignature(types.NewEIP155Signer(ei.chainId), common.Hex2Bytes(signedHash))
	if err := ei.client.SendTransaction(context.Background(), signedTx); err != nil {
		return nil, err
	}
	return signedTx.Hash().Bytes(), nil
}

func (ei *EvmToken) SendWrap(txid string, amount *big.Int, to string) ([]byte, error) {
	var data []byte
	data = append(data, MethodWrap[:4]...)
	data = append(data, common.Hex2Bytes(txid)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	value := big.NewInt(0)

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Get SuggestGasPrice error. %v", err)
	}

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.address)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), ei.privateKey)
	if err != nil {
		return nil, err
	}

	err = ei.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx.Hash().Bytes(), nil
}

func (ei *EvmToken) SendUnWrap(txid string, amount *big.Int, to string) ([]byte, error) {
	var data []byte
	data = append(data, MethodUnWrap[:4]...)
	data = append(data, common.Hex2Bytes(txid)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	value := big.NewInt(0)

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Get SuggestGasPrice error. %v", err)
	}

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.address)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), ei.privateKey)
	if err != nil {
		return nil, err
	}

	err = ei.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx.Hash().Bytes(), nil
}

func (ei *EvmToken) CreateUnWrapTxData(addr string, amount *big.Int, extra []byte) ([]byte, []byte, error) {
	return nil, nil, fmt.Errorf("Don't support this method")
}
func (ei *EvmToken) ValiditeUnWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	return tokens.BaseTransaction{}, fmt.Errorf("Don't support this method")
}
