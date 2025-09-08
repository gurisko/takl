package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage TAKL daemon",
	Long: `Control the TAKL background daemon that provides fast operations via Unix socket.

The daemon runs in the background and provides:
- Fast indexed search
- Project health monitoring
- Background file watching
- Caching for improved performance`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the TAKL daemon",
	Long: func() string {
		cfg := daemon.DefaultConfig()
		return fmt.Sprintf(`Start the TAKL daemon in foreground mode.

The daemon will:
- Listen on Unix socket at %s
- Monitor registered projects
- Provide fast cached operations

For background operation, use:
  nohup takl daemon start > /tmp/takl-daemon.log 2>&1 &
  
For production, use systemd or launchd service files in contrib/

Environment variables:
  TAKL_SOCKET - Override socket path (default: ~/.takl/daemon.sock)
  TAKL_PID    - Override PID file path (default: ~/.takl/daemon.pid)`, cfg.SocketPath)
	}(),
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

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the TAKL daemon",
	Long:  "Stop and then start the TAKL daemon.",
	RunE:  restartDaemon,
}

var daemonReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload daemon configuration",
	Long: `Reload the TAKL daemon configuration without restarting.

This triggers a live reload of:
- Project configurations
- Paradigm settings and options
- WIP limits and workflow guards
- Notification settings

The daemon will validate the new configuration and apply changes
gracefully. If validation fails, the old configuration remains active.`,
	RunE: reloadDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonReloadCmd)
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
	client := sdkClient()

	status, err := client.GetDaemonStatus()
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	if status.Running {
		fmt.Println("Daemon is running")
		if status.Uptime != "" {
			fmt.Printf("Uptime: %s\n", status.Uptime)
		}
		if status.RequestCount > 0 {
			fmt.Printf("Total requests: %d\n", status.RequestCount)
		}
		if status.ProjectCount > 0 {
			fmt.Printf("Projects: %d\n", status.ProjectCount)
		}
	} else {
		fmt.Println("Daemon is not running")
	}

	return nil
}

func restartDaemon(cmd *cobra.Command, args []string) error {
	d, err := daemon.New(daemon.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize daemon: %w", err)
	}

	// Stop if running
	if d.IsRunning() {
		fmt.Println("Stopping daemon...")
		if err := d.Stop(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
	}

	// Start again
	fmt.Println("Starting daemon...")
	return d.Start()
}

func reloadDaemon(cmd *cobra.Command, args []string) error {
	client := sdkClient()

	err := client.ReloadConfig()
	if err != nil {
		return fmt.Errorf("failed to reload daemon configuration: %w", err)
	}

	fmt.Println("Daemon configuration reloaded successfully")
	return nil
}
