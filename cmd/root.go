package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rpcURL     string
	outputJSON bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "evm-tracer",
	Short: "Advanced EVM transaction debugger and gas optimizer",
	Long: `EVM Tracer is an advanced transaction debugging tool that uses Geth as a library
to analyze transactions and identify gas optimization opportunities.

It provides detailed insights into:
- Storage access patterns (SLOAD/SSTORE)
- Memory operations and expansion
- External calls and their gas usage
- Expensive operations
- Gas consumption by opcode
- Specific optimization recommendations`,
	Version: "1.0.0",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rpcURL, "rpc", "http://localhost:8545", "Ethereum RPC URL")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
}
