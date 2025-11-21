package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/devlongs/evm-tracer/internal/analyzer"
	"github.com/devlongs/evm-tracer/internal/formatter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var traceCmd = &cobra.Command{
	Use:   "trace [transaction-hash]",
	Short: "Trace a transaction and analyze gas optimization opportunities",
	Long: `Traces an Ethereum transaction using a custom EVM tracer and provides
detailed analysis of gas usage and optimization opportunities.

The transaction must be on the connected network (default: local node).

Example:
  evm-tracer trace 0x1234...
  evm-tracer trace 0x1234... --rpc https://mainnet.infura.io/v3/YOUR-KEY
  evm-tracer trace 0x1234... --json > report.json`,
	Args: cobra.ExactArgs(1),
	RunE: runTrace,
}

func runTrace(cmd *cobra.Command, args []string) error {
	txHashStr := args[0]

	// Validate transaction hash
	if !common.IsHexAddress(txHashStr) && len(txHashStr) != 66 {
		return fmt.Errorf("invalid transaction hash: %s", txHashStr)
	}

	txHash := common.HexToHash(txHashStr)

	if verbose {
		fmt.Printf("🔍 Analyzing transaction: %s\n", txHash.Hex())
		fmt.Printf("📡 Connecting to: %s\n\n", rpcURL)
	}

	// Create analyzer
	an, err := analyzer.NewTransactionAnalyzer(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}
	defer an.Close()

	// Analyze transaction
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if verbose {
		fmt.Println("⚙️  Tracing transaction...")
	}

	err = an.AnalyzeTransaction(ctx, txHash)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Get results
	tracer := an.GetTracer()

	// Output results
	if outputJSON {
		report, err := tracer.GetReport()
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}
		fmt.Println(formatter.FormatJSON(report))
	} else {
		// Get optimizations
		optimizations := tracer.GetOptimizations()

		// Format and display
		output := formatter.FormatOptimizations(optimizations, tracer.TotalGasUsed)
		fmt.Print(output)

		// Show gas breakdown if verbose
		if verbose {
			breakdown := formatter.FormatGasBreakdown(tracer.GasPerOpcode, tracer.TotalGasUsed)
			fmt.Print(breakdown)
		}

		// Summary recommendations
		if len(optimizations) > 0 {
			fmt.Println("💡 RECOMMENDATIONS:")
			fmt.Println("   1. Review high-priority optimizations first")
			fmt.Println("   2. Consider caching frequently accessed storage values")
			fmt.Println("   3. Batch external calls when possible")
			fmt.Println("   4. Use memory instead of storage for temporary data")
			fmt.Println()
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(traceCmd)
}
