package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/hostfile"
	"github.com/kamrify/localname/internal/portfwd"
	"github.com/spf13/cobra"
)

var startPort int

var startCmd = &cobra.Command{
	Use:   "start [name] --port [port]",
	Short: "Start proxying a domain",
	Long: `Map a .local domain to a local port and start proxying.
Runs first-time setup automatically if needed.

  localname start myapp --port 3000
  # https://myapp.local → localhost:3000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := normalizeName(args[0])

		if err := config.ValidateDomain(name, startPort); err != nil {
			return err
		}

		if err := ensureSetup(); err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.SetDomain(name, startPort); err != nil {
			return err
		}

		if err := hostfile.Add(name); err != nil {
			return fmt.Errorf("updating /etc/hosts: %w", err)
		}

		if err := cert.EnsureLeafCert(name); err != nil {
			return fmt.Errorf("generating certificate: %w", err)
		}

		if !daemon.IsRunning() {
			if err := daemon.RunDetached(); err != nil {
				return fmt.Errorf("starting daemon: %w", err)
			}
			if err := daemon.WaitForDaemon(); err != nil {
				return err
			}
		} else {
			daemon.SendIPC(daemon.Request{Type: daemon.MsgReload})
		}

		fmt.Printf("https://%s.local → localhost:%d\n", name, startPort)
		return nil
	},
}

func ensureSetup() error {
	if !cert.CAExists() {
		fmt.Print("First-time setup: generating root CA... ")
		if err := cert.GenerateCA(); err != nil {
			return err
		}
		fmt.Println("done")

		fmt.Println("Trusting root CA (you may be prompted for your password)...")
		if err := cert.TrustCA(); err != nil {
			return fmt.Errorf("trusting CA: %w", err)
		}
	}

	pf := portfwd.New()
	if !pf.IsEnabled() {
		fmt.Print("Setting up port forwarding (80→10080, 443→10443)... ")
		if err := pf.Enable(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}
	}

	return nil
}

func init() {
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Local port to proxy to (required)")
	startCmd.MarkFlagRequired("port")
	rootCmd.AddCommand(startCmd)
}
