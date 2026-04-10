package charm

import (
	"fmt"
	"strings"
)

// detectSnippet is a shell snippet that resolves the default-route
// network interface and falls back to eth0 when detection fails.
const detectSnippet = `iface=$(ip -o route show default | awk '{print $5}' | head -n1); iface=${iface:-eth0}; `

// buildApplyCommand builds a command that applies tc netem rules.
// When iface is empty, the command auto-detects the default interface.
func buildApplyCommand(cfg *Config, iface string) []string {
	tcArgs := buildNetemArgs(cfg)

	if iface != "" {
		args := []string{"tc", "qdisc", "add", "dev", iface, "root", "netem"}
		return append(args, tcArgs...)
	}

	script := detectSnippet + "tc qdisc add dev $iface root netem " + strings.Join(tcArgs, " ")
	return []string{"sh", "-c", script}
}

// buildCleanupCommand builds a command that removes tc netem rules.
// When iface is empty, the command auto-detects the default interface.
func buildCleanupCommand(iface string) []string {
	if iface != "" {
		return []string{"tc", "qdisc", "del", "dev", iface, "root"}
	}

	script := detectSnippet + "tc qdisc del dev $iface root"
	return []string{"sh", "-c", script}
}

func buildNetemArgs(cfg *Config) []string {
	var args []string

	if cfg.Latency > 0 {
		args = append(args, "delay", formatDuration(cfg.Latency))

		if cfg.Jitter > 0 {
			args = append(args, formatDuration(cfg.Jitter))
		}
	}

	if cfg.PacketLoss > 0 {
		args = append(args, "loss", formatPercent(cfg.PacketLoss))
	}

	if cfg.Corruption > 0 {
		args = append(args, "corrupt", formatPercent(cfg.Corruption))
	}

	return args
}

func formatDuration(d fmt.Stringer) string {
	text := d.String()

	for _, suffix := range []struct{ long, short string }{
		{"µs", "us"},
	} {
		text = strings.ReplaceAll(text, suffix.long, suffix.short)
	}

	return text
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}
