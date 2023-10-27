package tokens

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var EventUnWrapFailed = crypto.Keccak256Hash([]byte("UnWrapFailed(bytes32,bytes32,bytes32)"))
var EventWrapFailed = crypto.Keccak256Hash([]byte("WrapFailed(bytes32,bytes32,bytes32)"))
var MethodGetFailTxes = crypto.Keccak256Hash([]byte("getFailTxes(bytes32)"))

type TxErrorRecordContract struct {
	client     *ethclient.Client
	rpc        string
	wss        string
	chainId    *big.Int
	account    common.Address
	contract   common.Address
	ListenType int           //0: listen event, 1: scan block
	TimePeriod time.Duration //only for scan block
}

func NewTxErrorRecordContract(_rpc, _wss, conAddr string, _listenType int, seconds time.Duration) (*TxErrorRecordContract, error) {
	c, err := ethclient.Dial(_rpc)
	if err != nil {
		return nil, err
	}
	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	return &TxErrorRecordContract{
		rpc:        _rpc,
		wss:        _wss,
		client:     c,
		chainId:    chainId,
		account:    common.HexToAddress("0x96216849c49358B10257cb55b28eA603c874b05E"),
		contract:   common.HexToAddress(conAddr),
		ListenType: _listenType,
		TimePeriod: seconds * time.Second,
	}, err
}

func (c *TxErrorRecordContract) StartListen(ch chan *TxErrorRecord) {
	if c.ListenType == ListenEvent {
		c.listenEvent(ch)
	} else if c.ListenType == ScanBlock {
		c.scanBlock(ch)
	}
}

func (c *TxErrorRecordContract) scanBlock(ch chan *TxErrorRecord) {
	fromHeight, err := c.client.BlockNumber(context.Background())
	if err != nil {
		errOrder := &TxErrorRecord{
			Type:  0,
			Error: fmt.Errorf("get the block number error. %v", err),
		}
		ch <- errOrder
		return
	}

	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.contract},
		Topics:    [][]common.Hash{{EventUnWrapFailed, EventWrapFailed}},
	}

	for {
		time.Sleep(c.TimePeriod)
		var toHeight uint64
		if toHeight, err = c.client.BlockNumber(context.Background()); err != nil {
			errOrder := &TxErrorRecord{
				Type:  1,
				Error: fmt.Errorf("BlockNumber error. %v", err),
			}
			ch <- errOrder
			continue
		} else if toHeight < fromHeight {
			continue
		}

		query.FromBlock = new(big.Int).SetUint64(fromHeight)
		query.ToBlock = new(big.Int).SetUint64(toHeight)
		logs, err := c.client.FilterLogs(context.Background(), query)
		if err != nil {
			errOrder := &TxErrorRecord{
				Type:  1,
				Error: fmt.Errorf("client FilterLogs error. %v", err),
			}
			ch <- errOrder
			continue
		}
		for i := range logs {
			if logs[i].Removed {
				continue
			}
			if bytes.Equal(logs[i].Topics[0][:], EventUnWrapFailed[:]) {
				c.dealFailedEvent(-1, &logs[i], ch)
			} else if bytes.Equal(logs[i].Topics[0][:], EventWrapFailed[:]) {
				c.dealFailedEvent(1, &logs[i], ch)
			}
		}
		fromHeight = toHeight + 1
	}
}

func (c *TxErrorRecordContract) listenEvent(ch chan *TxErrorRecord) {
	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.contract},
		Topics:    [][]common.Hash{{EventUnWrapFailed, EventWrapFailed}},
	}

	errOrder := &TxErrorRecord{Type: 0}

	//Create the ethclient
	client, err := ethclient.Dial(c.wss)
	if err != nil {
		errOrder.Error = fmt.Errorf("the EthWssClient redial error. %v. The EthWssClient will be redialed later", err)
		ch <- errOrder
		return
	}
	eventLogChan := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, eventLogChan)
	if err != nil || sub == nil {
		errOrder.Error = fmt.Errorf("get event logs from eth wss client error. %v", err)
		ch <- errOrder
		return
	}
	for {
		select {
		case err := <-sub.Err():
			errOrder.Error = fmt.Errorf("event wss sub error. %v. The EthWssClient will be redialed later", err)
			ch <- errOrder
			return
		case vLog := <-eventLogChan:
			if vLog.Removed {
				continue
			}
			if bytes.Equal(vLog.Topics[0][:], EventUnWrapFailed[:]) {
				c.dealFailedEvent(-1, &vLog, ch)
			} else if bytes.Equal(vLog.Topics[0][:], EventWrapFailed[:]) {
				c.dealFailedEvent(1, &vLog, ch)
			}
		}
	}
}

func (c *TxErrorRecordContract) dealFailedEvent(d int, vLog *types.Log, ch chan *TxErrorRecord) {
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) != 96 {
		ch <- &TxErrorRecord{
			Type:  0,
			Error: fmt.Errorf("error order record event data is error. %s, %s", tx, vLog.Address.Hex()),
		}
		return
	}
	txid := vLog.Data[:32]
	fromCoin, _, _ := bytes.Cut(vLog.Data[32:64], []byte{0})
	toCoin, _, _ := bytes.Cut(vLog.Data[64:96], []byte{0})
	ter := &TxErrorRecord{
		D:        d,
		Txid:     txid,
		FromCoin: string(fromCoin),
		ToCoin:   string(toCoin),
	}

	ch <- ter
}
