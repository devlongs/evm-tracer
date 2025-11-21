package analyzer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/devlongs/evm-tracer/internal/tracer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// TransactionAnalyzer handles the analysis of transactions
type TransactionAnalyzer struct {
	client *ethclient.Client
	tracer *tracer.GasOptimizationTracer
}

// NewTransactionAnalyzer creates a new transaction analyzer
func NewTransactionAnalyzer(rpcURL string) (*TransactionAnalyzer, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	return &TransactionAnalyzer{
		client: client,
		tracer: tracer.NewGasOptimizationTracer(),
	}, nil
}

// AnalyzeTransaction analyzes a transaction and returns optimization opportunities
func (a *TransactionAnalyzer) AnalyzeTransaction(ctx context.Context, txHash common.Hash) error {
	// Get transaction
	tx, pending, err := a.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}
	if pending {
		return fmt.Errorf("transaction is still pending")
	}

	// Get receipt
	receipt, err := a.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to get receipt: %w", err)
	}

	// Get block
	block, err := a.client.BlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	// Find transaction index in block
	txIndex := 0
	for i, blockTx := range block.Transactions() {
		if blockTx.Hash() == txHash {
			txIndex = i
			break
		}
	}

	// Create state database for the block
	statedb, err := a.createStateDB(ctx, block, txIndex)
	if err != nil {
		return fmt.Errorf("failed to create state: %w", err)
	}

	// Get message from transaction
	msg, err := core.TransactionToMessage(tx, types.LatestSignerForChainID(tx.ChainId()), block.BaseFee())
	if err != nil {
		return fmt.Errorf("failed to convert tx to message: %w", err)
	}

	// Create EVM context
	blockContext := core.NewEVMBlockContext(block.Header(), a, nil)
	txContext := core.NewEVMTxContext(msg)

	// Create EVM with our custom tracer
	vmConfig := vm.Config{
		Tracer:    a.tracer,
		NoBaseFee: false,
	}

	evm := vm.NewEVM(blockContext, txContext, statedb, params.MainnetChainConfig, vmConfig)

	// Execute the transaction
	_, err = core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(block.GasLimit()))
	if err != nil {
		// Even if execution fails, we might have useful trace data
		fmt.Printf("Transaction execution error (this is OK for analysis): %v\n", err)
	}

	return nil
}

// createStateDB creates a state database for analysis
// This is a simplified version - in production, you'd need proper state access
func (a *TransactionAnalyzer) createStateDB(ctx context.Context, block *types.Block, txIndex int) (*state.StateDB, error) {
	// Note: This requires an archive node for proper historical state access
	// For simplicity, we create a new in-memory state
	db := rawdb.NewMemoryDatabase()
	statedb, err := state.New(block.Root(), state.NewDatabase(db), nil)
	if err != nil {
		return nil, err
	}
	return statedb, nil
}

// GetTracer returns the tracer instance
func (a *TransactionAnalyzer) GetTracer() *tracer.GasOptimizationTracer {
	return a.tracer
}

// Close closes the analyzer connection
func (a *TransactionAnalyzer) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// GetHeaderByNumber implements ChainContext interface
func (a *TransactionAnalyzer) GetHeader(hash common.Hash, number uint64) *types.Header {
	header, err := a.client.HeaderByNumber(context.Background(), big.NewInt(int64(number)))
	if err != nil {
		return nil
	}
	return header
}

// GetHeaderByHash implements ChainContext interface
func (a *TransactionAnalyzer) GetHeaderByHash(hash common.Hash) *types.Header {
	// This is used by the EVM during execution
	return nil
}

// Engine implements ChainContext interface
func (a *TransactionAnalyzer) Engine() consensus.Engine {
	return nil
}
