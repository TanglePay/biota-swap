package evm

import (
	"bwrap/gl"
	"bwrap/tokens"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func (ei *EvmToken) StartListen(ch chan *tokens.SwapOrder) {
	if ei.ListenType == 0 {
		ei.listenEvent(ch)
	} else if ei.ListenType == 1 {
		ei.scanBlock(ch)
	}
}

func (ei *EvmToken) scanBlock(ch chan *tokens.SwapOrder) {
	fromHeight, err := ei.client.BlockNumber(context.Background())
	if err != nil {
		errOrder := &tokens.SwapOrder{
			Type:  0,
			Error: fmt.Errorf("Get the block number error. %v", err),
		}
		ch <- errOrder
		return
	}

	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
		Topics:    [][]common.Hash{{EventUnWrap}},
	}

	for {
		time.Sleep(10 * time.Second)
		var toHeight uint64
		if toHeight, err = ei.client.BlockNumber(context.Background()); err != nil {
			errOrder := &tokens.SwapOrder{
				Type:  1,
				Error: fmt.Errorf(" error. %v", err),
			}
			ch <- errOrder
			continue
		} else if toHeight <= fromHeight {
			continue
		}

		query.FromBlock = new(big.Int).SetUint64(fromHeight)
		query.ToBlock = new(big.Int).SetUint64(toHeight)
		logs, err := ei.client.FilterLogs(context.Background(), query)
		if err != nil {
			errOrder := &tokens.SwapOrder{
				Type:  1,
				Error: fmt.Errorf("client FilterLogs error. %v", err),
			}
			ch <- errOrder
			continue
		}
		for i := range logs {
			ei.dealTransferEvent(ch, &logs[i])
		}
		fromHeight = toHeight + 1
	}
}

func (ei *EvmToken) listenEvent(ch chan *tokens.SwapOrder) {
	nodeUrl := "wss://" + ei.url
	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
		Topics:    [][]common.Hash{{EventUnWrap}},
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
			ei.dealTransferEvent(ch, &vLog)
		}
	}
}

func (ei *EvmToken) dealTransferEvent(ch chan *tokens.SwapOrder, vLog *types.Log) {
	errOrder := &tokens.SwapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("UnWrap event data is nil. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	symbol := string(vLog.Data[:32])
	amount := new(big.Int).SetBytes(vLog.Data[32:])
	account := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	gl.OutLogger.Info("UnWrap token. %s : %s : %s", tx, account, amount.String())

	order := &tokens.SwapOrder{
		TxID:      tx,
		FromToken: ei.Symbol(),
		ToToken:   symbol,
		From:      common.BytesToAddress(vLog.Topics[1][:]).Hex(),
		To:        common.Bytes2Hex(vLog.Topics[2][:]),
		Amount:    amount,
		Error:     nil,
	}
	ch <- order
}
