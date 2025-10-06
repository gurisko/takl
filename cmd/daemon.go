package cmd

import (
	"fmt"
	"time"

	"github.com/gurisko/takl/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage TAKL daemon",
	Long: `Control the TAKL background daemon that provides fast operations via Unix socket.

The daemon runs in the background and provides:
- HTTP API over Unix socket
- Fast operations
- Background processing`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the TAKL daemon",
	Long: `Start the TAKL daemon in foreground mode.

For background operation, use:
  nohup takl daemon start > /tmp/takl-daemon.log 2>&1 &`,
	RunE: startDaemon,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the TAKL daemon",
	Long:  "Stop the running TAKL daemon gracefully.",
	RunE:  stopDaemon,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  "Check if the TAKL daemon is running and display its status.",
	RunE:  statusDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}

func startDaemon(cmd *cobra.Command, args []string) error {
	d, err := daemon.New(daemon.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize daemon: %w", err)
	}

	return d.Start()
}

func stopDaemon(cmd *cobra.Command, args []string) error {
	d, err := daemon.New(daemon.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize daemon: %w", err)
	}

	return d.Stop()
}

func statusDaemon(cmd *cobra.Command, args []string) error {
	d, err := daemon.New(daemon.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize daemon: %w", err)
	}

	status, err := d.GetStatus()
	if err != nil {
		return err
	}

	// Format for display
	if !status.Running {
		if status.PID > 0 {
			if status.ErrorMessage != "" {
				fmt.Printf("TAKL daemon process exists (PID: %d) but not responding\n", status.PID)
				fmt.Printf("  Socket: %s\n", status.SocketPath)
				fmt.Printf("  Error: %v\n", status.ErrorMessage)
			} else {
				fmt.Printf("TAKL daemon is not running (stale pidfile)\n")
				fmt.Printf("  Socket: %s\n", status.SocketPath)
			}
		} else {
			fmt.Printf("TAKL daemon is not running\n")
			fmt.Printf("  Socket: %s\n", status.SocketPath)
		}
	} else {
		fmt.Printf("TAKL daemon running (PID: %d)\n", status.PID)
		fmt.Printf("  Socket: %s\n", status.SocketPath)
		fmt.Printf("  Uptime: %s\n", status.Uptime.Round(time.Second))
	}

	return nil
}
