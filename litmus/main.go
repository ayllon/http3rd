package main

import (
	"github.com/ayllon/http3rd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gitlab.cern.ch/flutter/go-proxy"
)

var (
	debug  bool
	params http3rd.Params
)

// Return the user certificate and private key to use
// If flagCert is not set, it will try to figure it out
func setupUserCredentials(params *http3rd.Params) {
	if params.UserCert != "" {
		if params.UserKey == "" {
			params.UserKey = params.UserCert
		}
		return
	}
	var e error
	params.UserCert, params.UserKey, e = proxy.GetCertAndKeyLocation()
	if e != nil {
		logrus.Fatal(e)
	}
}

var rootCmd = &cobra.Command{
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		setupUserCredentials(&params)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var testCmd = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.BoolVar(&debug, "debug", false, "Enable debug output")
	flags.StringVar(&params.CAPath, "capath", "/etc/grid-security/certificates", "CA Path")
	flags.StringVar(&params.UserCert, "cert", "", "User certificate")
	flags.StringVar(&params.UserKey, "key", "", "User private key")
	flags.BoolVar(&params.Insecure, "insecure", false, "Do not verify the remote certificate")

	rootCmd.AddCommand(testCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
