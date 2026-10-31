package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
		return runWebServer()
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
		return runWebServer()
	},
}

func runWebServer() error {
	webHostInst, token, err := appInst.CreateWebHost()
	if err != nil {
		return fmt.Errorf("failed to initialize web host: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", webHost, webPort)
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

func init() {
	webCmd.Flags().StringVar(&webHost, "host", "127.0.0.1", "Web server host address")
	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Web server port")

	serverStartCmd.Flags().StringVar(&webHost, "host", "127.0.0.1", "Web server host address")
	serverStartCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Web server port")

	serverCmd.AddCommand(serverStartCmd)

	RootCmd.AddCommand(webCmd)
	RootCmd.AddCommand(serverCmd)
}
