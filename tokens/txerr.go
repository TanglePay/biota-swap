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

var EventUnWrapFailed = crypto.Keccak256Hash([]byte("UnWrapFailed(bytes32,bytes32,bytes32,uint256)"))
var EventWrapFailed = crypto.Keccak256Hash([]byte("WrapFailed(bytes32,bytes32,bytes32,uint256)"))
var MethodGetFailTxes = crypto.Keccak256Hash([]byte("getFailTxes(bytes32)"))

type TxErrorRecordContract struct {
	client     *ethclient.Client
	url        string
	chainId    *big.Int
	account    common.Address
	contract   common.Address
	ListenType int //0: listen event, 1: scan block
	TimePeriod time.Duration
}

func NewTxErrorRecordContract(uri, conAddr string, _listenType int, seconds time.Duration) (*TxErrorRecordContract, error) {
	c, err := ethclient.Dial("https://" + uri)
	if err != nil {
		return nil, err
	}
	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	return &TxErrorRecordContract{
		url:        uri,
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
	} else if c.ListenType == ScanBlock {
		c.scanBlock(ch)
	}
}

func (c *TxErrorRecordContract) scanBlock(ch chan *TxErrorRecord) {
	fromHeight, err := c.client.BlockNumber(context.Background())
	if err != nil {
		errOrder := &TxErrorRecord{
			Type:  0,
			Error: fmt.Errorf("Get the block number error. %v", err),
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
			if bytes.Compare(logs[i].Topics[0][:], EventUnWrapFailed[:]) == 0 {
				c.dealUnWrapFailed(&logs[i], ch)
			} else if bytes.Compare(logs[i].Topics[0][:], EventWrapFailed[:]) == 0 {
				c.dealUnWrapFailed(&logs[i], ch)
			}
		}
		fromHeight = toHeight + 1
	}
}

func (c *TxErrorRecordContract) listenEvent(ch chan *TxErrorRecord) {
	nodeUrl := "wss://" + c.url
	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.contract},
		Topics:    [][]common.Hash{{EventUnWrapFailed, EventWrapFailed}},
	}

	errOrder := &TxErrorRecord{Type: 0}

	//Create the ethclient
	client, err := ethclient.Dial(nodeUrl)
	if err != nil {
		errOrder.Error = fmt.Errorf("The EthWssClient redial error. %v\nThe EthWssClient will be redialed later...\n", err)
		ch <- errOrder
		return
	}
	eventLogChan := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, eventLogChan)
	if err != nil || sub == nil {
		errOrder.Error = fmt.Errorf("Get event logs from eth wss client error. %v\n", err)
		ch <- errOrder
		return
	}
	for {
		select {
		case err := <-sub.Err():
			errOrder.Error = fmt.Errorf("Event wss sub error. %v\nThe EthWssClient will be redialed later...\n", err)
			ch <- errOrder
			return
		case vLog := <-eventLogChan:
			if vLog.Removed {
				continue
			}
			if vLog.Topics[0].Hex() == "" {
				c.dealUnWrapFailed(&vLog, ch)
			} else {
				c.dealUnWrapFailed(&vLog, ch)
			}
		}
	}
}

func (c *TxErrorRecordContract) dealUnWrapFailed(vLog *types.Log, ch chan *TxErrorRecord) {
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) != 128 {
		ch <- &TxErrorRecord{
			Type:  0,
			Error: fmt.Errorf("error order record event data is error. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex()),
		}
		return
	}
	txid := vLog.Data[:32]
	fromCoin, _, _ := bytes.Cut(vLog.Data[32:64], []byte{0})
	toCoin, _, _ := bytes.Cut(vLog.Data[64:96], []byte{0})
	amount := new(big.Int).SetBytes(vLog.Data[96:])
	ter := &TxErrorRecord{
		txid:     txid,
		fromCoin: string(fromCoin),
		toCoin:   string(toCoin),
		amount:   amount,
	}
	data := make([]byte, 0)
	data = append(data, MethodGetFailTxes[:4]...)
	data = append(data, ter.txid...)
	msg := ethereum.CallMsg{From: c.account, To: &c.contract, Data: data}
	result, err := c.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		ch <- &TxErrorRecord{
			Type:  0,
			Error: fmt.Errorf("CallContract error. %v", err),
		}
		return
	}
	count := new(big.Int).SetBytes(result[32:64]).Int64()
	data = result[64:]
	failedTxes := make([][]byte, 0)
	for i := int64(0); i < count; i++ {
		failedTxes = append(failedTxes, data[:32])
		data = data[32:]
	}
	ter.failedTxes = failedTxes

	ch <- ter
}
