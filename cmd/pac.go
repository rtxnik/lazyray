package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var (
	pacPort      int
	pacFile      string
	pacSystemOpt bool
)

var pacCmd = &cobra.Command{
	Use:   "pac",
	Short: "PAC file generation and serving",
	Long:  "Generate or serve a PAC (Proxy Auto-Configuration) file so browsers can auto-select the lazyray proxy. Use the subcommands to print a PAC file or to serve it over HTTP.",
}

var pacGenerateCmd = &cobra.Command{
	Use:     "generate",
	Short:   "Generate PAC file to stdout or file",
	Long:    "Generate a PAC (Proxy Auto-Configuration) file for the default proxy profile and print it to stdout, or write it to a file with --output. Point a browser's auto-config URL at the resulting file to route matching traffic through lazyray.",
	Example: "  lzr pac generate\n  lzr pac generate --output proxy.pac",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		profile := servers.DefaultProfile()
		content := core.GeneratePAC(settings, profile)

		if pacFile != "" {
			if err := os.WriteFile(pacFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("writing PAC file: %w", err)
			}
			fmt.Printf("PAC file written to %s\n", pacFile)
			return nil
		}

		fmt.Print(content)
		return nil
	},
}

var pacServeCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Serve PAC file over HTTP",
	Long:    "Start a local HTTP server that serves the PAC file for the default proxy profile. Point a browser's automatic proxy-configuration URL at it, or pass --system to also set that PAC URL as the system proxy (auto-rolled-back when the server stops).",
	Example: "  lzr pac serve\n  lzr pac serve --port 8080\n  lzr pac serve --system",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		profile := servers.DefaultProfile()
		content := core.GeneratePAC(settings, profile)

		mux := http.NewServeMux()
		mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
			w.Header().Set("Cache-Control", "no-cache")
			fmt.Fprint(w, content)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, "lazyray PAC server\nUse: http://127.0.0.1:%d/proxy.pac\n", pacPort)
		})

		addr := fmt.Sprintf("127.0.0.1:%d", pacPort)
		pacURL := fmt.Sprintf("http://%s/proxy.pac", addr)

		fmt.Printf("Serving PAC file at %s\n", pacURL)

		// Set system proxy to PAC URL if --system flag is set
		if pacSystemOpt {
			sp := platform.CurrentSystemProxy()
			if err := sp.EnablePACProxy(pacURL); err != nil {
				return fmt.Errorf("setting system PAC proxy: %w", err)
			}
			fmt.Println("System proxy configured to use PAC URL")
		} else {
			fmt.Printf("Configure your browser proxy auto-config URL to: %s\n", pacURL)
		}

		fmt.Println("Press Ctrl+C to stop.")

		server := &http.Server{Addr: addr, Handler: mux}

		// Handle graceful shutdown with signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\nShutting down PAC server...")

			// Rollback system proxy settings
			if pacSystemOpt {
				sp := platform.CurrentSystemProxy()
				if err := sp.Disable(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to disable system proxy: %v\n", err)
				} else {
					fmt.Println("System proxy settings restored")
				}
			}

			_ = server.Shutdown(context.Background())
		}()

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// Rollback on unexpected error too
			if pacSystemOpt {
				_ = platform.CurrentSystemProxy().Disable()
			}
			return err
		}
		return nil
	},
}

func init() {
	pacGenerateCmd.Flags().StringVarP(&pacFile, "output", "o", "", "write the PAC file to this path instead of stdout")
	pacServeCmd.Flags().IntVarP(&pacPort, "port", "p", 10810, "Port to serve PAC file on")
	pacServeCmd.Flags().BoolVar(&pacSystemOpt, "system", false, "Set PAC URL as system proxy (auto-rollback on stop)")

	pacCmd.AddCommand(pacGenerateCmd)
	pacCmd.AddCommand(pacServeCmd)
	rootCmd.AddCommand(pacCmd)
}
