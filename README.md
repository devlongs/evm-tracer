# EVM Tracer

Advanced transaction debugger and gas optimizer using Geth as a library.

## Features

- **Custom EVM Tracer**: Implements `vm.EVMLogger` to track opcode execution
- **Gas Optimization Detection**: Identifies redundant operations and expensive patterns
- **Deep Analysis**: Storage access, memory operations, external calls, per-opcode gas usage
- **CLI Interface**: Color-coded output with severity levels and JSON export

## Installation

```bash
git clone https://github.com/devlongs/evm-tracer
cd evm-tracer
go build -o evm-tracer .
```

Or use the Makefile:

```bash
make build
```

## Usage

```bash
# Basic usage
./evm-tracer trace 0xTRANSACTION_HASH

# With custom RPC
./evm-tracer trace 0xTX_HASH --rpc https://mainnet.infura.io/v3/YOUR_KEY

# Verbose output with gas breakdown
./evm-tracer trace 0xTX_HASH --verbose

# JSON export
./evm-tracer trace 0xTX_HASH --json > report.json
```

## Example Output

```
═══════════════════════════════════════════════════════════════
           EVM TRACER - GAS OPTIMIZATION REPORT
═══════════════════════════════════════════════════════════════

📊 Total Gas Used: 125.43K
🔍 Optimizations Found: 4

🚨 HIGH PRIORITY OPTIMIZATIONS

1. redundant_sload
   Description: Multiple SLOAD operations for the same storage slot
   💰 Potential Savings: 300 gas
   Details:
     • read_count: 4
     • storage_key: 0xabcd...

💰 Total Potential Savings: 2.10K (~1.67%)
```

## Architecture

```
cmd/              CLI commands (root, trace)
internal/
  tracer/         Custom EVM tracer implementation
  analyzer/       Transaction replay and Geth integration
  formatter/      Output formatting (console, JSON)
```

### How It Works

1. **Tracer** implements `vm.EVMLogger` interface to hook into EVM execution
2. **Analyzer** fetches transaction data and replays it with the custom tracer
3. **Formatter** presents findings with color-coded severity levels

## Detected Optimizations

**High Priority**
- Redundant SLOAD operations (~100 gas/read)
- Repeated storage writes to same slot (~2,900+ gas)

**Medium Priority**
- Expensive opcodes (CREATE, KECCAK256, LOG)
- Multiple external calls (batch for ~2,100 gas savings)
- Memory expansion (quadratic cost)

**Low Priority**
- Inefficient gas forwarding patterns

## Testing

```bash
# Run tests
go test ./internal/tracer/ -v

# Test with local node
npx hardhat node  # Terminal 1
./evm-tracer trace 0xTX_HASH  # Terminal 2

# Test with mainnet
./evm-tracer trace 0xTX_HASH --rpc https://mainnet.infura.io/v3/YOUR_KEY
```

## Requirements

- Go 1.21+
- Ethereum RPC node (local or remote)

## License

MIT - see [LICENSE](LICENSE)
