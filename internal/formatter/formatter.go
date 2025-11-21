package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/devlongs/evm-tracer/internal/tracer"
	"github.com/fatih/color"
)

var (
	highSeverity   = color.New(color.FgRed, color.Bold)
	mediumSeverity = color.New(color.FgYellow, color.Bold)
	lowSeverity    = color.New(color.FgCyan)
	successColor   = color.New(color.FgGreen, color.Bold)
	headerColor    = color.New(color.FgMagenta, color.Bold)
	infoColor      = color.New(color.FgWhite)
)

// FormatOptimizations formats optimization results for console output
func FormatOptimizations(optimizations []tracer.Optimization, totalGas uint64) string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n"))
	sb.WriteString(headerColor.Sprint("           EVM TRACER - GAS OPTIMIZATION REPORT\n"))
	sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n\n"))

	// Summary
	sb.WriteString(infoColor.Sprintf("📊 Total Gas Used: %s\n", formatGas(totalGas)))
	sb.WriteString(infoColor.Sprintf("🔍 Optimizations Found: %d\n\n", len(optimizations)))

	if len(optimizations) == 0 {
		sb.WriteString(successColor.Sprint("✨ No obvious optimization opportunities found!\n"))
		sb.WriteString(successColor.Sprint("   Your transaction appears to be well-optimized.\n\n"))
		return sb.String()
	}

	// Group by severity
	high := []tracer.Optimization{}
	medium := []tracer.Optimization{}
	low := []tracer.Optimization{}

	for _, opt := range optimizations {
		switch opt.Severity {
		case "high":
			high = append(high, opt)
		case "medium":
			medium = append(medium, opt)
		case "low":
			low = append(low, opt)
		}
	}

	// Display by severity
	if len(high) > 0 {
		sb.WriteString(highSeverity.Sprint("🚨 HIGH PRIORITY OPTIMIZATIONS\n"))
		sb.WriteString(strings.Repeat("─", 63) + "\n")
		for i, opt := range high {
			sb.WriteString(formatOptimization(opt, i+1, "high"))
		}
		sb.WriteString("\n")
	}

	if len(medium) > 0 {
		sb.WriteString(mediumSeverity.Sprint("⚠️  MEDIUM PRIORITY OPTIMIZATIONS\n"))
		sb.WriteString(strings.Repeat("─", 63) + "\n")
		for i, opt := range medium {
			sb.WriteString(formatOptimization(opt, i+1, "medium"))
		}
		sb.WriteString("\n")
	}

	if len(low) > 0 {
		sb.WriteString(lowSeverity.Sprint("ℹ️  LOW PRIORITY OPTIMIZATIONS\n"))
		sb.WriteString(strings.Repeat("─", 63) + "\n")
		for i, opt := range low {
			sb.WriteString(formatOptimization(opt, i+1, "low"))
		}
		sb.WriteString("\n")
	}

	// Calculate total potential savings
	totalSavings := uint64(0)
	for _, opt := range optimizations {
		totalSavings += opt.GasSavings
	}

	if totalSavings > 0 {
		sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n"))
		sb.WriteString(successColor.Sprintf("💰 Total Potential Savings: %s (~%.2f%%)\n",
			formatGas(totalSavings),
			float64(totalSavings)/float64(totalGas)*100))
		sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n\n"))
	}

	return sb.String()
}

func formatOptimization(opt tracer.Optimization, index int, severity string) string {
	var sb strings.Builder
	var severityColor *color.Color

	switch severity {
	case "high":
		severityColor = highSeverity
	case "medium":
		severityColor = mediumSeverity
	case "low":
		severityColor = lowSeverity
	}

	sb.WriteString(severityColor.Sprintf("\n%d. %s\n", index, opt.Type))
	sb.WriteString(fmt.Sprintf("   Description: %s\n", opt.Description))
	sb.WriteString(fmt.Sprintf("   Location: %s\n", opt.Location))

	if opt.GasSavings > 0 {
		sb.WriteString(fmt.Sprintf("   💰 Potential Savings: %s\n", formatGas(opt.GasSavings)))
	}

	if len(opt.Details) > 0 {
		sb.WriteString("   Details:\n")

		// Sort keys for consistent output
		keys := make([]string, 0, len(opt.Details))
		for k := range opt.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := opt.Details[key]
			sb.WriteString(fmt.Sprintf("     • %s: %v\n", key, value))
		}
	}

	return sb.String()
}

// FormatGasBreakdown formats gas usage by opcode
func FormatGasBreakdown(gasPerOpcode map[string]uint64, totalGas uint64) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n"))
	sb.WriteString(headerColor.Sprint("                    GAS USAGE BREAKDOWN\n"))
	sb.WriteString(headerColor.Sprint("═══════════════════════════════════════════════════════════════\n\n"))

	// Sort opcodes by gas usage
	type opcodeGas struct {
		opcode string
		gas    uint64
	}

	opcodes := make([]opcodeGas, 0, len(gasPerOpcode))
	for op, gas := range gasPerOpcode {
		opcodes = append(opcodes, opcodeGas{op, gas})
	}

	sort.Slice(opcodes, func(i, j int) bool {
		return opcodes[i].gas > opcodes[j].gas
	})

	// Show top 10 gas consumers
	limit := 10
	if len(opcodes) < limit {
		limit = len(opcodes)
	}

	sb.WriteString(fmt.Sprintf("%-20s %15s %10s\n", "OPCODE", "GAS USED", "% OF TOTAL"))
	sb.WriteString(strings.Repeat("─", 63) + "\n")

	for i := 0; i < limit; i++ {
		op := opcodes[i]
		percentage := float64(op.gas) / float64(totalGas) * 100

		colorFunc := infoColor
		if percentage > 20 {
			colorFunc = highSeverity
		} else if percentage > 10 {
			colorFunc = mediumSeverity
		}

		sb.WriteString(colorFunc.Sprintf("%-20s %15s %9.2f%%\n",
			op.opcode,
			formatGas(op.gas),
			percentage))
	}

	sb.WriteString("\n")
	return sb.String()
}

func formatGas(gas uint64) string {
	if gas >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(gas)/1000000)
	} else if gas >= 1000 {
		return fmt.Sprintf("%.2fK", float64(gas)/1000)
	}
	return fmt.Sprintf("%d", gas)
}

// FormatJSON formats the trace as JSON
func FormatJSON(report string) string {
	return report
}
