package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running pidge server",
	Long:  "Send SIGTERM to the running pidge serve process for a graceful shutdown.",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return fmt.Errorf("reading /proc: %w", err)
	}

	myPID := os.Getpid()
	found := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == myPID {
			continue
		}

		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}

		parts := strings.Split(string(cmdline), "\x00")
		if len(parts) >= 2 && strings.HasSuffix(parts[0], "pidge") && parts[1] == "serve" {
			proc, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("sending SIGTERM to pid %d: %w", pid, err)
			}
			fmt.Printf("Sent SIGTERM to pidge serve (pid %d)\n", pid)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("no running pidge serve process found")
	}
	return nil
}
