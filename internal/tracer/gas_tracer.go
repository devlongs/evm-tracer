package tracer

import (
	"encoding/json"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// GasOptimizationTracer is a custom tracer that tracks gas optimization opportunities
type GasOptimizationTracer struct {
	mu sync.Mutex

	// Tracking data
	StorageReads  map[common.Hash]int  // Track repeated SLOAD operations
	StorageWrites map[common.Hash]int  // Track SSTORE operations
	MemoryOps     []MemoryOperation    // Track memory operations
	CallOps       []CallOperation      // Track call operations
	Loops         []LoopDetection      // Detect potential loops
	ExpensiveOps  []ExpensiveOperation // Track expensive operations
	GasPerOpcode  map[string]uint64    // Gas used per opcode

	// Current state
	Stack        []uint256 // Current stack state
	Memory       []byte    // Current memory
	PC           uint64    // Program counter
	Gas          uint64    // Remaining gas
	Depth        int       // Call depth
	TotalGasUsed uint64    // Total gas used

	// Analysis results
	Optimizations []Optimization // Identified optimizations
}

type MemoryOperation struct {
	PC    uint64
	Op    string
	Size  uint64
	Gas   uint64
	Depth int
}

type CallOperation struct {
	PC      uint64
	Op      string
	To      common.Address
	Value   *big.Int
	Gas     uint64
	GasUsed uint64
	Success bool
	Depth   int
}

type LoopDetection struct {
	StartPC    uint64
	EndPC      uint64
	Iterations int
	GasPerLoop uint64
}

type ExpensiveOperation struct {
	PC          uint64
	Op          string
	Gas         uint64
	Description string
	Depth       int
}

type Optimization struct {
	Type        string
	Severity    string // "high", "medium", "low"
	Description string
	Location    string
	GasSavings  uint64
	Details     map[string]interface{}
}

type uint256 [32]byte

// NewGasOptimizationTracer creates a new gas optimization tracer
func NewGasOptimizationTracer() *GasOptimizationTracer {
	return &GasOptimizationTracer{
		StorageReads:  make(map[common.Hash]int),
		StorageWrites: make(map[common.Hash]int),
		MemoryOps:     make([]MemoryOperation, 0),
		CallOps:       make([]CallOperation, 0),
		Loops:         make([]LoopDetection, 0),
		ExpensiveOps:  make([]ExpensiveOperation, 0),
		GasPerOpcode:  make(map[string]uint64),
		Optimizations: make([]Optimization, 0),
		Stack:         make([]uint256, 0),
	}
}

// CaptureStart implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Gas = gas
	t.Depth = 0
}

// CaptureState implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.PC = pc
	t.Gas = gas
	t.Depth = depth
	t.TotalGasUsed += cost

	opName := op.String()
	t.GasPerOpcode[opName] += cost

	// Track storage operations
	switch op {
	case vm.SLOAD:
		// Check if we have data on stack (we can't directly check len, so use Back with error handling)
		key := scope.Stack.Back(0)
		if key != nil {
			keyHash := common.BytesToHash(key.Bytes())
			t.StorageReads[keyHash]++

			// Check for redundant SLOADs
			if t.StorageReads[keyHash] > 2 {
				t.Optimizations = append(t.Optimizations, Optimization{
					Type:        "redundant_sload",
					Severity:    "high",
					Description: "Multiple SLOAD operations for the same storage slot",
					Location:    formatPC(pc),
					GasSavings:  (uint64(t.StorageReads[keyHash]) - 1) * 100, // SLOAD warm cost ~100 gas
					Details: map[string]interface{}{
						"storage_key": keyHash.Hex(),
						"read_count":  t.StorageReads[keyHash],
					},
				})
			}
		}

	case vm.SSTORE:
		key := scope.Stack.Back(0)
		if key != nil {
			keyHash := common.BytesToHash(key.Bytes())
			t.StorageWrites[keyHash]++
		}

	case vm.MLOAD, vm.MSTORE, vm.MSTORE8:
		t.MemoryOps = append(t.MemoryOps, MemoryOperation{
			PC:    pc,
			Op:    opName,
			Size:  uint64(len(scope.Memory.Data())),
			Gas:   cost,
			Depth: depth,
		})

	case vm.CALL, vm.STATICCALL, vm.DELEGATECALL, vm.CALLCODE:
		callOp := CallOperation{
			PC:      pc,
			Op:      opName,
			Gas:     gas,
			GasUsed: cost,
			Depth:   depth,
		}

		gasLimit := scope.Stack.Back(0)
		addr := scope.Stack.Back(1)
		if gasLimit != nil && addr != nil {
			callOp.To = common.BytesToAddress(addr.Bytes())

			// Check for inefficient gas forwarding
			if gasLimit.Uint64() == gas-gas/64 {
				t.Optimizations = append(t.Optimizations, Optimization{
					Type:        "gas_forwarding",
					Severity:    "low",
					Description: "Forwarding all available gas to external call",
					Location:    formatPC(pc),
					GasSavings:  0,
					Details: map[string]interface{}{
						"call_type": opName,
						"to":        callOp.To.Hex(),
					},
				})
			}
		}

		t.CallOps = append(t.CallOps, callOp)

	case vm.CREATE, vm.CREATE2:
		t.ExpensiveOps = append(t.ExpensiveOps, ExpensiveOperation{
			PC:          pc,
			Op:          opName,
			Gas:         cost,
			Description: "Contract creation is expensive",
			Depth:       depth,
		})

	case vm.SELFDESTRUCT:
		t.ExpensiveOps = append(t.ExpensiveOps, ExpensiveOperation{
			PC:          pc,
			Op:          opName,
			Gas:         cost,
			Description: "SELFDESTRUCT is very expensive",
			Depth:       depth,
		})

	case vm.JUMPDEST:
		// Track potential loops
		// Simple heuristic: if we see the same JUMPDEST multiple times in quick succession
		// This is a simplified loop detection

	case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
		if cost > 1000 {
			t.ExpensiveOps = append(t.ExpensiveOps, ExpensiveOperation{
				PC:          pc,
				Op:          opName,
				Gas:         cost,
				Description: "Large LOG operation",
				Depth:       depth,
			})
		}

	case vm.KECCAK256:
		if cost > 500 {
			t.ExpensiveOps = append(t.ExpensiveOps, ExpensiveOperation{
				PC:          pc,
				Op:          opName,
				Gas:         cost,
				Description: "Expensive KECCAK256 operation",
				Depth:       depth,
			})
		}
	}

	// Track memory expansion
	if len(scope.Memory.Data()) > 0 {
		memSize := uint64(len(scope.Memory.Data()))
		if memSize > 10000 {
			t.Optimizations = append(t.Optimizations, Optimization{
				Type:        "memory_expansion",
				Severity:    "medium",
				Description: "Large memory expansion detected",
				Location:    formatPC(pc),
				GasSavings:  0,
				Details: map[string]interface{}{
					"memory_size": memSize,
				},
			})
		}
	}
}

// CaptureEnter implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Depth++
}

// CaptureExit implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Depth--
	t.TotalGasUsed += gasUsed
}

// CaptureFault implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Track faults for analysis
}

// CaptureEnd implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TotalGasUsed = gasUsed

	// Final analysis
	t.analyzePatterns()
}

// CaptureTxStart implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureTxStart(gasLimit uint64) {
	t.Gas = gasLimit
}

// CaptureTxEnd implements the EVMLogger interface
func (t *GasOptimizationTracer) CaptureTxEnd(restGas uint64) {
	// Transaction ended
}

// analyzePatterns performs final analysis to identify optimization patterns
func (t *GasOptimizationTracer) analyzePatterns() {
	// Analyze opcode usage
	for opcode, gasUsed := range t.GasPerOpcode {
		if gasUsed > t.TotalGasUsed/10 { // If opcode uses >10% of total gas
			t.Optimizations = append(t.Optimizations, Optimization{
				Type:        "expensive_opcode",
				Severity:    "medium",
				Description: "Opcode consumes significant gas",
				Location:    "multiple",
				GasSavings:  0,
				Details: map[string]interface{}{
					"opcode":     opcode,
					"gas_used":   gasUsed,
					"percentage": float64(gasUsed) / float64(t.TotalGasUsed) * 100,
				},
			})
		}
	}

	// Analyze call patterns
	if len(t.CallOps) > 5 {
		t.Optimizations = append(t.Optimizations, Optimization{
			Type:        "multiple_calls",
			Severity:    "medium",
			Description: "Multiple external calls detected - consider batching",
			Location:    "multiple",
			GasSavings:  uint64(len(t.CallOps)) * 2100, // Base call cost savings
			Details: map[string]interface{}{
				"call_count": len(t.CallOps),
			},
		})
	}
}

// GetOptimizations returns all identified optimizations
func (t *GasOptimizationTracer) GetOptimizations() []Optimization {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.Optimizations
}

// GetReport generates a JSON report of the trace
func (t *GasOptimizationTracer) GetReport() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	report := map[string]interface{}{
		"total_gas_used":    t.TotalGasUsed,
		"storage_reads":     len(t.StorageReads),
		"storage_writes":    len(t.StorageWrites),
		"memory_operations": len(t.MemoryOps),
		"call_operations":   len(t.CallOps),
		"expensive_ops":     len(t.ExpensiveOps),
		"optimizations":     t.Optimizations,
		"gas_by_opcode":     t.GasPerOpcode,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func formatPC(pc uint64) string {
	return "0x" + common.Bytes2Hex(big.NewInt(int64(pc)).Bytes())
}
