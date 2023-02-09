package smrevm

import (
	"biota_swap/gl"
	"biota_swap/tokens"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (ei *EvmIota) StartListen(ch chan *tokens.SwapOrder) {
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

func (ei *EvmIota) dealTransferEvent(ch chan *tokens.SwapOrder, vLog *types.Log) {
	errOrder := &tokens.SwapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("UnWrap event data is nil. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	data := new(big.Int).SetBytes(vLog.Data)
	account := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	gl.OutLogger.Info("UnWrap token. %s : %s : %s", tx, account, data.String())

	order := &tokens.SwapOrder{
		TxID:   tx,
		From:   common.BytesToAddress(vLog.Topics[1][:]).Hex(),
		To:     common.Bytes2Hex(vLog.Topics[2][:]),
		Amount: data.String(),
		Error:  nil,
	}
	ch <- order
}
