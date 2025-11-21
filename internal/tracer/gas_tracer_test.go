package tracer

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewGasOptimizationTracer(t *testing.T) {
	tracer := NewGasOptimizationTracer()

	if tracer == nil {
		t.Fatal("Expected tracer to be created")
	}

	if tracer.StorageReads == nil {
		t.Error("StorageReads map not initialized")
	}

	if tracer.StorageWrites == nil {
		t.Error("StorageWrites map not initialized")
	}

	if tracer.GasPerOpcode == nil {
		t.Error("GasPerOpcode map not initialized")
	}
}

func TestGetOptimizations(t *testing.T) {
	tracer := NewGasOptimizationTracer()

	// Initially should have no optimizations
	opts := tracer.GetOptimizations()
	if len(opts) != 0 {
		t.Errorf("Expected 0 optimizations, got %d", len(opts))
	}

	// Add a test optimization
	tracer.Optimizations = append(tracer.Optimizations, Optimization{
		Type:        "test",
		Severity:    "high",
		Description: "Test optimization",
		Location:    "0x42",
		GasSavings:  100,
	})

	opts = tracer.GetOptimizations()
	if len(opts) != 1 {
		t.Errorf("Expected 1 optimization, got %d", len(opts))
	}

	if opts[0].Type != "test" {
		t.Errorf("Expected type 'test', got '%s'", opts[0].Type)
	}
}

func TestFormatPC(t *testing.T) {
	tests := []struct {
		pc       uint64
		expected string
	}{
		{0, "0x"},
		{42, "0x2a"},
		{255, "0xff"},
		{256, "0x0100"},
	}

	for _, tt := range tests {
		result := formatPC(tt.pc)
		if result != tt.expected {
			t.Errorf("formatPC(%d) = %s, expected %s", tt.pc, result, tt.expected)
		}
	}
}

func TestStorageTracking(t *testing.T) {
	tracer := NewGasOptimizationTracer()

	// Simulate storage reads
	key := common.HexToHash("0x1234")
	tracer.StorageReads[key] = 1
	tracer.StorageReads[key]++
	tracer.StorageReads[key]++

	if tracer.StorageReads[key] != 3 {
		t.Errorf("Expected 3 reads, got %d", tracer.StorageReads[key])
	}
}

func TestGetReport(t *testing.T) {
	tracer := NewGasOptimizationTracer()
	tracer.TotalGasUsed = 50000
	tracer.GasPerOpcode["SLOAD"] = 2100
	tracer.GasPerOpcode["SSTORE"] = 5000

	report, err := tracer.GetReport()
	if err != nil {
		t.Fatalf("GetReport() error: %v", err)
	}

	if report == "" {
		t.Error("Expected non-empty report")
	}

	// Check if report contains expected fields
	if !contains(report, "total_gas_used") {
		t.Error("Report missing 'total_gas_used'")
	}

	if !contains(report, "gas_by_opcode") {
		t.Error("Report missing 'gas_by_opcode'")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsRecursive(s, substr)))
}

func containsRecursive(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
