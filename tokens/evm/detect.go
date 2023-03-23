package evm

import (
	"bwrap/tokens"
	"bwrap/tools/crypto"
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var EventUnWrap = crypto.Keccak256Hash([]byte("UnWrap(address,bytes32,bytes32,uint256)"))
var EventWrap = crypto.Keccak256Hash([]byte("Wrap(address,address,bytes32,uint256)"))

func (ei *EvmToken) StartWrapListen(ch chan *tokens.SwapOrder) {
	if ei.ListenType == tokens.ListenEvent {
		ei.listenEvent(EventWrap, ch)
	} else if ei.ListenType == tokens.ScanBlock {
		ei.scanBlock(EventWrap, ch)
	}
}

func (ei *EvmToken) StartUnWrapListen(ch chan *tokens.SwapOrder) {
	if ei.ListenType == tokens.ListenEvent {
		ei.listenEvent(EventUnWrap, ch)
	} else if ei.ListenType == tokens.ScanBlock {
		ei.scanBlock(EventUnWrap, ch)
	}
}

func (ei *EvmToken) scanBlock(event common.Hash, ch chan *tokens.SwapOrder) {
	fromHeight, err := ei.client.BlockNumber(context.Background())
	if err != nil {
		errOrder := &tokens.SwapOrder{
			Type:  0,
			Error: fmt.Errorf("Get the block number error. %s : %v", ei.symbol, err),
		}
		ch <- errOrder
		return
	}

	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
		Topics:    [][]common.Hash{{event}},
	}

	for {
		time.Sleep(10 * time.Second)
		var toHeight uint64
		if toHeight, err = ei.client.BlockNumber(context.Background()); err != nil {
			errOrder := &tokens.SwapOrder{
				Type:  1,
				Error: fmt.Errorf("BlockNumber error. %s : %v", ei.symbol, err),
			}
			ch <- errOrder
			continue
		} else if toHeight < fromHeight {
			continue
		}

		query.FromBlock = new(big.Int).SetUint64(fromHeight)
		query.ToBlock = new(big.Int).SetUint64(toHeight)
		logs, err := ei.client.FilterLogs(context.Background(), query)
		if err != nil {
			errOrder := &tokens.SwapOrder{
				Type:  1,
				Error: fmt.Errorf("client FilterLogs error. %s : %v", ei.symbol, err),
			}
			ch <- errOrder
			continue
		}
		for i := range logs {
			if logs[i].Topics[0].Hex() == EventWrap.Hex() {
				ei.dealWrapEvent(ch, &logs[i])
			} else {
				ei.dealUnWrapEvent(ch, &logs[i])
			}
		}
		fromHeight = toHeight + 1
	}
}

func (ei *EvmToken) listenEvent(event common.Hash, ch chan *tokens.SwapOrder) {
	nodeUrl := "wss://" + ei.url
	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
		Topics:    [][]common.Hash{{event}},
	}

	errOrder := &tokens.SwapOrder{Type: 0}

	//Create the ethclient
	c, err := ethclient.Dial(nodeUrl)
	if err != nil {
		errOrder.Error = fmt.Errorf("The EthWssClient redial error. %v\nThe EthWssClient will be redialed later...\n", err)
		ch <- errOrder
		return
	}
	eventLogChan := make(chan types.Log)
	sub, err := c.SubscribeFilterLogs(context.Background(), query, eventLogChan)
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
			if vLog.Topics[0].Hex() == EventWrap.Hex() {
				ei.dealWrapEvent(ch, &vLog)
			} else {
				ei.dealUnWrapEvent(ch, &vLog)
			}
		}
	}
}

func (ei *EvmToken) dealWrapEvent(ch chan *tokens.SwapOrder, vLog *types.Log) {
	errOrder := &tokens.SwapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) != 64 {
		errOrder.Error = fmt.Errorf("Wrap event data is nil. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	fromAddr := common.BytesToAddress(vLog.Topics[1][:]).Hex()
	toAddr := common.BytesToAddress(vLog.Topics[2][:]).Hex()
	symbol, _, _ := bytes.Cut(vLog.Data[:32], []byte{0})
	amount := new(big.Int).SetBytes(vLog.Data[32:])

	order := &tokens.SwapOrder{
		TxID:      tx,
		FromToken: ei.Symbol(),
		ToToken:   string(symbol),
		From:      fromAddr,
		To:        toAddr,
		Amount:    amount,
		Error:     nil,
	}
	ch <- order
}

func (ei *EvmToken) dealUnWrapEvent(ch chan *tokens.SwapOrder, vLog *types.Log) {
	errOrder := &tokens.SwapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("UnWrap event data is nil. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	symbol, _, _ := bytes.Cut(vLog.Data[:32], []byte{0})
	amount := new(big.Int).SetBytes(vLog.Data[32:])

	order := &tokens.SwapOrder{
		TxID:      tx,
		FromToken: ei.Symbol(),
		ToToken:   string(symbol),
		From:      common.BytesToAddress(vLog.Topics[1][:]).Hex(),
		To:        common.Bytes2Hex(vLog.Topics[2][:]),
		Amount:    amount,
		Error:     nil,
	}
	ch <- order
}
