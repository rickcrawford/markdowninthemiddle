package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/rickcrawford/markdowninthemiddle/internal/certs"
)

var gencertCmd = &cobra.Command{
	Use:   "gencert",
	Short: "Generate a self-signed TLS certificate",
	Long:  `Creates a self-signed ECDSA P-256 TLS certificate and writes cert.pem and key.pem to the specified directory.`,
	RunE:  runGencert,
}

func init() {
	gencertCmd.Flags().String("host", "localhost", "hostname or IP for the certificate")
	gencertCmd.Flags().String("dir", "./certs", "output directory for cert and key files")
	rootCmd.AddCommand(gencertCmd)
}

func runGencert(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	dir, _ := cmd.Flags().GetString("dir")

	certPath, keyPath, err := certs.Generate(host, dir)
	if err != nil {
		return fmt.Errorf("generating certificate: %w", err)
	}

	log.Printf("certificate written to %s", certPath)
	log.Printf("private key written to %s", keyPath)
	return nil
}
