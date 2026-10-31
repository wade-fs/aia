package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	webHost string
	webPort int
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the AIA embedded Web UI and API server",
	Long: `Start the single-binary AIA Embedded Web UI and REST/SSE API server.
On startup, a cryptographically random one-time login token is generated and printed.
Use this token to log in from your browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWebServer(cmd)
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage the AIA server process",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the AIA server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWebServer(cmd)
	},
}

func runWebServer(cmd *cobra.Command) error {
	host := webHost
	if !cmd.Flags().Changed("host") {
		if envH := os.Getenv("AIA_HOST"); envH != "" {
			host = envH
		}
	}

	port := webPort
	if !cmd.Flags().Changed("port") {
		if envP := os.Getenv("AIA_PORT"); envP != "" {
			if p, err := strconv.Atoi(envP); err == nil && p > 0 {
				port = p
			}
		}
	}

	webHostInst, token, err := appInst.CreateWebHost()
	if err != nil {
		return fmt.Errorf("failed to initialize web host: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      webHostInst.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("=======================================================================")
	fmt.Printf("🚀 AIA Embedded Web Console running at: http://%s/\n", addr)
	fmt.Println("=======================================================================")
	fmt.Printf("🔑 One-Time Login Token: \n\n   %s\n\n", token)
	fmt.Println("👉 Note: This token can only be redeemed ONCE. Keep it confidential.")
	fmt.Println("=======================================================================")

	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	// Background ticker for scheduled tasks and due items
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-bgCtx.Done():
				return
			case <-ticker.C:
				if appInst.ScheduleSvc != nil {
					_, _ = appInst.ScheduleSvc.ProcessDueSchedules(bgCtx)
				}
				if appInst.Scheduler != nil {
					_, _ = appInst.Scheduler.ProcessReadyTasks(bgCtx)
				}
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-sigCh:
		fmt.Println("\nShutting down web server gracefully...")
		bgCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

func init() {
	defaultHost := "127.0.0.1"
	if h := os.Getenv("AIA_HOST"); h != "" {
		defaultHost = h
	}
	defaultPort := 7777
	if p := os.Getenv("AIA_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			defaultPort = v
		}
	}

	webCmd.Flags().StringVar(&webHost, "host", defaultHost, "Web server host address (or env $AIA_HOST)")
	webCmd.Flags().IntVarP(&webPort, "port", "p", defaultPort, "Web server port (or env $AIA_PORT)")

	serverStartCmd.Flags().StringVar(&webHost, "host", defaultHost, "Web server host address (or env $AIA_HOST)")
	serverStartCmd.Flags().IntVarP(&webPort, "port", "p", defaultPort, "Web server port (or env $AIA_PORT)")

	serverCmd.AddCommand(serverStartCmd)

	RootCmd.AddCommand(webCmd)
	RootCmd.AddCommand(serverCmd)
}
