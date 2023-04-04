package zksync2

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"time"
)

type Provider interface {
	GetClient() *ethclient.Client
	GetBalance(address common.Address, blockNumber BlockNumber) (*big.Int, error)
	GetBlockByNumber(blockNumber BlockNumber) (*Block, error)
	GetBlockByHash(blockHash common.Hash) (*Block, error)
	GetTransactionCount(address common.Address, blockNumber BlockNumber) (*big.Int, error)
	GetTransactionReceipt(txHash common.Hash) (*TransactionReceipt, error)
	GetTransaction(txHash common.Hash) (*TransactionResponse, error)
	WaitMined(ctx context.Context, txHash common.Hash) (*TransactionReceipt, error)
	WaitFinalized(ctx context.Context, txHash common.Hash) (*TransactionReceipt, error)
	EstimateGas(tx *Transaction) (*big.Int, error)
	GetGasPrice() (*big.Int, error)
	SendRawTransaction(tx []byte) (common.Hash, error)
	ZksGetMainContract() (common.Address, error)
	ZksL1ChainId() (*big.Int, error)
	ZksL1BatchNumber() (*big.Int, error)
	ZksGetConfirmedTokens(from uint32, limit uint8) ([]*Token, error)
	ZksIsTokenLiquid(address common.Address) (bool, error)
	ZksGetTokenPrice(address common.Address) (*big.Float, error)
	ZksGetL2ToL1LogProof(txHash common.Hash, logIndex int) (*L2ToL1MessageProof, error)
	ZksGetL2ToL1MsgProof(block uint32, sender common.Address, msg common.Hash) (*L2ToL1MessageProof, error)
	ZksGetAllAccountBalances(address common.Address) (map[common.Address]*big.Int, error)
	ZksGetBridgeContracts() (*BridgeContracts, error)
	ZksEstimateFee(tx *Transaction) (*Fee, error)
	ZksGetTestnetPaymaster() (common.Address, error)
	ZksGetBlockDetails(block uint32) (*BlockDetails, error)
	GetLogs(q FilterQuery) ([]Log, error)
}

func NewDefaultProvider(rawUrl string) (*DefaultProvider, error) {
	rpcClient, err := rpc.Dial(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to rpc.Dial(): %w", err)
	}
	return &DefaultProvider{
		c:      rpcClient,
		Client: ethclient.NewClient(rpcClient),
	}, nil
}

type DefaultProvider struct {
	c *rpc.Client
	// also inherit default Ethereum client
	*ethclient.Client
}

func (p *DefaultProvider) GetClient() *ethclient.Client {
	return p.Client
}

func (p *DefaultProvider) GetBalance(address common.Address, blockNumber BlockNumber) (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "eth_getBalance", address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getBalance: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) GetBlockByNumber(blockNumber BlockNumber) (*Block, error) {
	type TmpBlock struct {
		Number           hexutil.Big  `json:"number"`
		L1BatchNumber    *hexutil.Big `json:"l1BatchNumber"`
		L1BatchTimestamp *hexutil.Big `json:"l1BatchTimestamp"`
	}
	var resp *TmpBlock
	err := p.c.Call(&resp, "eth_getBlockByNumber", blockNumber, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getBlockByNumber: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	ethBlock, err := p.Client.BlockByNumber(context.Background(), resp.Number.ToInt())
	if err != nil {
		return nil, err
	}
	return &Block{
		Block:            *ethBlock,
		L1BatchNumber:    resp.L1BatchNumber,
		L1BatchTimestamp: resp.L1BatchTimestamp,
	}, nil
}

func (p *DefaultProvider) GetBlockByHash(blockHash common.Hash) (*Block, error) {
	type TmpBlock struct {
		L1BatchNumber    *hexutil.Big `json:"l1BatchNumber"`
		L1BatchTimestamp *hexutil.Big `json:"l1BatchTimestamp"`
	}
	var resp *TmpBlock
	err := p.c.Call(&resp, "eth_getBlockByHash", blockHash, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getBlockByHash: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	ethBlock, err := p.Client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return nil, err
	}
	return &Block{
		Block:            *ethBlock,
		L1BatchNumber:    resp.L1BatchNumber,
		L1BatchTimestamp: resp.L1BatchTimestamp,
	}, nil
}

func (p *DefaultProvider) GetTransactionCount(address common.Address, blockNumber BlockNumber) (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "eth_getTransactionCount", address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getTransactionCount: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) GetTransactionReceipt(txHash common.Hash) (*TransactionReceipt, error) {
	var resp *TransactionReceipt
	err := p.c.Call(&resp, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getTransactionReceipt: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	return resp, nil
}

func (p *DefaultProvider) GetTransaction(txHash common.Hash) (*TransactionResponse, error) {
	var resp *TransactionResponse
	err := p.c.Call(&resp, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_getTransactionByHash: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	return resp, nil
}

func (p *DefaultProvider) EstimateGas(tx *Transaction) (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "eth_estimateGas", tx, BlockNumberLatest)
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_estimateGas: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) GetGasPrice() (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "eth_gasPrice")
	if err != nil {
		return nil, fmt.Errorf("failed to query eth_gasPrice: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) SendRawTransaction(tx []byte) (common.Hash, error) {
	var res string
	err := p.c.Call(&res, "eth_sendRawTransaction", hexutil.Encode(tx))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to call eth_sendRawTransaction: %w", err)
	}
	return common.HexToHash(res), nil
}

func (p *DefaultProvider) ZksGetMainContract() (common.Address, error) {
	var res string
	err := p.c.Call(&res, "zks_getMainContract")
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to query zks_getMainContract: %w", err)
	}
	return common.HexToAddress(res), nil
}

func (p *DefaultProvider) ZksL1ChainId() (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "zks_L1ChainId")
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_L1ChainId: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) ZksL1BatchNumber() (*big.Int, error) {
	var res string
	err := p.c.Call(&res, "zks_L1BatchNumber")
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_L1BatchNumber: %w", err)
	}
	resp, err := hexutil.DecodeBig(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as big.Int: %w", err)
	}
	return resp, nil
}

func (p *DefaultProvider) ZksGetConfirmedTokens(from uint32, limit uint8) ([]*Token, error) {
	res := make([]*Token, 0)
	err := p.c.Call(&res, "zks_getConfirmedTokens", from, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getConfirmedTokens: %w", err)
	}
	return res, nil
}

func (p *DefaultProvider) ZksIsTokenLiquid(address common.Address) (bool, error) {
	var res bool
	err := p.c.Call(&res, "zks_isTokenLiquid", address)
	if err != nil {
		return false, fmt.Errorf("failed to query zks_isTokenLiquid: %w", err)
	}
	return res, nil
}

func (p *DefaultProvider) ZksGetTokenPrice(address common.Address) (*big.Float, error) {
	var res string
	err := p.c.Call(&res, "zks_getTokenPrice", address)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getTokenPrice: %w", err)
	}
	resp, ok := big.NewFloat(0).SetString(res)
	if !ok {
		return nil, errors.New("failed to decode response as big.Float")
	}
	return resp, nil
}

func (p *DefaultProvider) ZksGetL2ToL1LogProof(txHash common.Hash, logIndex int) (*L2ToL1MessageProof, error) {
	var resp *L2ToL1MessageProof
	err := p.c.Call(&resp, "zks_getL2ToL1LogProof", txHash, logIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getL2ToL1LogProof: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	return resp, nil
}

func (p *DefaultProvider) ZksGetL2ToL1MsgProof(block uint32, sender common.Address, msg common.Hash) (*L2ToL1MessageProof, error) {
	var resp *L2ToL1MessageProof
	err := p.c.Call(&resp, "zks_getL2ToL1MsgProof", block, sender, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getL2ToL1MsgProof: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	return resp, nil
}

func (p *DefaultProvider) ZksGetAllAccountBalances(address common.Address) (map[common.Address]*big.Int, error) {
	res := make(map[common.Address]string)
	err := p.c.Call(&res, "zks_getAllAccountBalances", address)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getAllAccountBalances: %w", err)
	}
	resp := make(map[common.Address]*big.Int, len(res))
	for t, b := range res {
		resp[t], err = hexutil.DecodeBig(b)
		if err != nil {
			return nil, fmt.Errorf("failed to decode one of balances as big.Int: %w", err)
		}
	}
	return resp, nil
}

func (p *DefaultProvider) ZksGetBridgeContracts() (*BridgeContracts, error) {
	res := BridgeContracts{}
	err := p.c.Call(&res, "zks_getBridgeContracts")
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getBridgeContracts: %w", err)
	}
	return &res, nil
}

func (p *DefaultProvider) ZksEstimateFee(tx *Transaction) (*Fee, error) {
	var res Fee
	err := p.c.Call(&res, "zks_estimateFee", tx)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_estimateFee: %w", err)
	}
	return &res, nil
}

func (p *DefaultProvider) ZksGetTestnetPaymaster() (common.Address, error) {
	var res string
	err := p.c.Call(&res, "zks_getTestnetPaymaster")
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to query zks_estimateFee: %w", err)
	}
	return common.HexToAddress(res), nil
}

func (p *DefaultProvider) ZksGetBlockDetails(block uint32) (*BlockDetails, error) {
	var resp *BlockDetails
	err := p.c.Call(&resp, "zks_getBlockDetails", block)
	if err != nil {
		return nil, fmt.Errorf("failed to query zks_getBlockDetails: %w", err)
	} else if resp == nil {
		return nil, ethereum.NotFound
	}
	return resp, nil
}

func (p *DefaultProvider) WaitMined(ctx context.Context, txHash common.Hash) (*TransactionReceipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()
	for {
		receipt, err := p.GetTransactionReceipt(txHash)
		if err == nil && receipt.BlockNumber != nil {
			return receipt, nil
		}
		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

func (p *DefaultProvider) WaitFinalized(ctx context.Context, txHash common.Hash) (*TransactionReceipt, error) {
	receipt, err := p.WaitMined(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for tx is mined: %w", err)
	}
	if receipt.BlockNumber == nil {
		return nil, errors.New("empty tx block number")
	}
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()
	var blockHead *types.Header
	for {
		err = p.c.CallContext(ctx, &blockHead, "eth_getBlockByNumber", BlockNumberFinalized, false)
		if err == nil && blockHead == nil {
			err = ethereum.NotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get finalized block: %w", err)
		}
		if blockHead.Number.Cmp(receipt.BlockNumber) >= 0 {
			return receipt, nil
		}
		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

func (p *DefaultProvider) GetLogs(q FilterQuery) ([]Log, error) {
	var result []Log
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	err = p.c.Call(&result, "eth_getLogs", arg)
	return result, err
}
