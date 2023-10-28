package evm

import (
	"bwrap/gl"
	"bwrap/tokens"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var MethodWrap = crypto.Keccak256Hash([]byte("wrap(bytes32,uint256,address)"))
var MethodSend = crypto.Keccak256Hash([]byte("send(bytes32,uint256,address)"))
var MethodUnWrap = crypto.Keccak256Hash([]byte("unWrap(bytes32,bytes32,uint256)"))

// wrap(address to, bytes32 symbol)
// wrap(address to, bytes32 symbol, uint256 amount)
// unWrap(bytes32 to, bytes32 symbol, uint256 amount)
var MethodUserWrapEth = crypto.Keccak256Hash([]byte("wrap(address,bytes32)"))
var MethodUserWrapErc20 = crypto.Keccak256Hash([]byte("wrap(address,bytes32,uint256)"))
var MethodUserUnWrap = crypto.Keccak256Hash([]byte("unWrap(bytes32,bytes32,uint256)"))

type EvmToken struct {
	client        *ethclient.Client
	rpc           string
	wss           string
	chainId       *big.Int
	symbol        string
	contract      common.Address
	account       common.Address
	ListenType    int //0: listen event, 1: scan block
	ScanMaxHeight uint64
	GasPriceUpper int64
}

func NewEvmToken(rpc, wss, conAddr, symbol string, _account common.Address, _listenType int, maxHeight uint64, gasPriceUpper int64) (*EvmToken, error) {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}
	if gasPriceUpper > 100 {
		return nil, fmt.Errorf("GasPriceUpper is over 100 : %d", gasPriceUpper)
	}

	return &EvmToken{
		rpc:           rpc,
		wss:           wss,
		client:        c,
		chainId:       chainId,
		symbol:        symbol,
		contract:      common.HexToAddress(conAddr),
		account:       _account,
		ListenType:    _listenType,
		ScanMaxHeight: maxHeight,
		GasPriceUpper: gasPriceUpper,
	}, err
}

func (ei *EvmToken) MultiSignType() int {
	return tokens.EvmMultiSign
}

func (ei *EvmToken) ChainID() string {
	return ei.chainId.String()
}

func (ei *EvmToken) Symbol() string {
	return ei.symbol
}

func (ei *EvmToken) PublicKey() []byte {
	return nil
}

func (ei *EvmToken) KeyType() string {
	return "EC256K1"
}

func (ei *EvmToken) Address() string {
	return ei.contract.Hex()
}

func (ei *EvmToken) CheckSentTx(txid []byte) (bool, error) {
	hash := common.BytesToHash(txid)
	ftx, err := ei.client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		return true, fmt.Errorf("transactionReceipt error. %v", err)
	}
	if ftx.Status == 0 { //failed
		return false, fmt.Errorf("tx sent error. %s", hash.Hex())
	}
	return true, nil
}

func (ei *EvmToken) CheckUserTx(txid []byte, toCoin string, d int) (string, string, *big.Int, error) {
	hash := common.BytesToHash(txid)
	tx, isPending, err := ei.client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return "", "", nil, fmt.Errorf("client.TransactionByHash error. %s, %v", hash.Hex(), err)
	}
	if isPending {
		return "", "", nil, fmt.Errorf("tx is pending status. %s", hash.Hex())
	}
	signer := types.NewEIP155Signer(tx.ChainId())
	from, err := signer.Sender(tx)
	if err != nil {
		return "", "", nil, fmt.Errorf("get from address from tx error. %s : %v", hash.Hex(), err)
	}

	data := tx.Data()
	usrD := 0
	if bytes.Equal(data[:4], MethodUserWrapEth[:4]) {
		usrD = 1
	} else if bytes.Equal(data[:4], MethodUserWrapErc20[:4]) {
		usrD = 1
	} else if bytes.Equal(data[:4], MethodUserUnWrap[:4]) {
		usrD = -1
	}
	if d != usrD {
		return "", "", nil, fmt.Errorf("d error. %d,%d", usrD, d)
	}
	data = data[4:]

	to := ""
	if d == 1 {
		to = common.BytesToAddress(data[:32]).Hex()
	} else {
		to = hex.EncodeToString(data[:32])
	}

	sy, _, _ := bytes.Cut(data[32:64], []byte{0})
	if string(sy) != toCoin {
		return "", "", nil, fmt.Errorf("symbol is not equal. %s :%s", string(data), toCoin)
	}

	amount := tx.Value()
	if amount.Uint64() == 0 {
		amount = new(big.Int).SetBytes(data[64:])
	}

	return from.Hex(), to, amount, nil
}

func (ei *EvmToken) CheckTxFailed(failedTx, txid []byte, to string, amount *big.Int, d int) error {
	c, err := rpc.Dial(ei.rpc)
	if err != nil {
		return fmt.Errorf("rpc.Dial error.  %v", err)
	}

	var r *Receipt
	txHash := common.BytesToHash(failedTx)
	err = c.CallContext(context.Background(), &r, "eth_getTransactionReceipt", txHash)
	if err == nil {
		if r == nil {
			return fmt.Errorf("failedTx not found. %s", txHash.Hex())
		}
	}
	if err != nil {
		return fmt.Errorf("eth_getTransactionReceipt error. %v", err)
	}

	if r.Status != 0 {
		return fmt.Errorf("tx(%s) success", txHash.Hex())
	}

	if !bytes.Equal(ei.account[:], r.From[:]) {
		return fmt.Errorf("the `from` address is not equal. %s : %s : %s", ei.account.Hex(), r.From.Hex(), txHash.Hex())
	}

	tx, isPending, err := ei.client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		return fmt.Errorf("client.TransactionByHash error. %s, %v", txHash.Hex(), err)
	}
	if isPending {
		return fmt.Errorf("tx is pending status. %s", txHash.Hex())
	}

	data := tx.Data()
	if d == -1 {
		if !bytes.Equal(data[:4], MethodSend[:4]) {
			return fmt.Errorf("tx is not send")
		}
	} else {
		if !bytes.Equal(data[:4], MethodWrap[:4]) {
			return fmt.Errorf("tx is not wrap")
		}
	}
	data = data[4:]

	if !bytes.Equal(data[:32], txid) {
		return fmt.Errorf("txid is not equal. %s : %s", hex.EncodeToString(data[:32]), hex.EncodeToString(txid))
	}

	a := new(big.Int).SetBytes(data[32:64])
	if a.Cmp(amount) != 0 {
		return fmt.Errorf("amounts are not equal. %s : %s", a.String(), amount.String())
	}

	if to != common.BytesToAddress(data[64:]).Hex() {
		return fmt.Errorf("to addresses are not equal. %s : %s", common.BytesToAddress(data[64:]).Hex(), to)
	}
	return nil
}

func (ei *EvmToken) CheckUnWrapTx(txid []byte, to, symbol string, amount *big.Int) error {
	hash := common.BytesToHash(txid)
	tx, isPending, err := ei.client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return fmt.Errorf("client.TransactionByHash error. %s, %v", hash.Hex(), err)
	}
	if isPending {
		return fmt.Errorf("tx is pending status. %s", hash.Hex())
	}

	data := tx.Data()
	if !bytes.Equal(data[:4], MethodUnWrap[:4]) {
		return fmt.Errorf("tx is not UnWrap")
	}
	data = data[4:]

	if !bytes.Equal(common.FromHex(to), data[:32]) {
		return fmt.Errorf("to address is not equal. %s : %s", to, hex.EncodeToString(data[:32]))
	}
	data = data[32:]

	sy, _, _ := bytes.Cut(data[:32], []byte{0})
	if string(sy) != symbol {
		return fmt.Errorf("symbol is not equal. %s :%s", string(data), symbol)
	}
	data = data[32:]

	a := new(big.Int).SetBytes(data)
	if a.Cmp(amount) < 0 {
		return fmt.Errorf("amount is bigger. %d : %d", amount.Uint64(), a.Uint64())
	}

	return nil
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

func (ei *EvmToken) SendWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(common.FromHex(to)) != 20 {
		return nil, fmt.Errorf("to address error. %s", to)
	}

	txHash := common.FromHex(txid)
	if len(txHash) > 32 {
		txHash = txHash[:32]
	}
	var data []byte
	data = append(data, MethodWrap[:4]...)
	data = append(data, common.LeftPadBytes(txHash, 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	value := big.NewInt(0)

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.account)
	if err != nil {
		return nil, err
	}

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get SuggestGasPrice error. %v", err)
	}
	gasPrice.Mul(gasPrice, big.NewInt(100+ei.GasPriceUpper))
	gasPrice.Div(gasPrice, big.NewInt(100))

	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), prv)
	if err != nil {
		return nil, err
	}

	err = ei.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx.Hash().Bytes(), nil
}

func (ei *EvmToken) SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(common.FromHex(to)) != 20 {
		return nil, fmt.Errorf("to address error. %s", to)
	}

	txHash := common.FromHex(txid)
	if len(txHash) > 32 {
		txHash = txHash[:32]
	}
	var data []byte
	data = append(data, MethodSend[:4]...)
	data = append(data, common.LeftPadBytes(txHash, 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	value := big.NewInt(0)

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.account)
	if err != nil {
		return nil, err
	}

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get SuggestGasPrice error. %v", err)
	}
	gasPrice.Mul(gasPrice, big.NewInt(100+ei.GasPriceUpper))
	gasPrice.Div(gasPrice, big.NewInt(100))

	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), prv)
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
	return nil, nil, fmt.Errorf("don't support this method")
}

func (ei *EvmToken) ValiditeUnWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	return tokens.BaseTransaction{}, fmt.Errorf("don't support this method")
}

func (ei *EvmToken) CheckPendingAndSpeedUp(txHash common.Hash, prv *ecdsa.PrivateKey) (common.Hash, error) {
	tx, isPending, err := ei.client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		return txHash, fmt.Errorf("get tx by hash error. %v", err)
	}

	if !isPending {
		return txHash, nil
	}

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return txHash, fmt.Errorf("get SuggestGasPrice error. %v", err)
	}
	gasPrice.Mul(gasPrice, big.NewInt(100+ei.GasPriceUpper+20))
	gasPrice.Div(gasPrice, big.NewInt(100))
	if gasPrice.Cmp(tx.GasPrice()) < 0 {
		return txHash, fmt.Errorf("gasPrice suggest is lower than old tx. %s < %s", gasPrice.String(), tx.GasPrice().String())
	}

	tx = types.NewTransaction(tx.Nonce(), *tx.To(), tx.Value(), tx.Gas(), gasPrice, tx.Data())

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), prv)
	if err != nil {
		return txHash, fmt.Errorf("sign tx error. %v", err)
	}

	err = ei.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return signedTx.Hash(), fmt.Errorf("send tx error. %v", err)
	}

	return signedTx.Hash(), nil
}

type Receipt struct {
	// Consensus fields: These fields are defined by the Yellow Paper
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Status uint64         `json:"status"`
}

// UnmarshalJSON unmarshals from JSON.
func (r *Receipt) UnmarshalJSON(input []byte) error {
	type Receipt struct {
		From   *common.Address `json:"from"`
		To     *common.Address `json:"to"`
		Status *hexutil.Uint64 `json:"status"`
	}
	var dec Receipt
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.From != nil {
		r.From = *dec.From
	}
	if dec.To != nil {
		r.To = *dec.To
	}
	if dec.Status != nil {
		r.Status = uint64(*dec.Status)
	}
	return nil
}
