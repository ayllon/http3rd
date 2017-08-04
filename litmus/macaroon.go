package main

import (
	"github.com/ayllon/http3rd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	macaroonLifetime = time.Minute
)

var macaroonCmd = &cobra.Command{
	Use: "macaroon <url> <activity1> [<activity2> [<activity3>]]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Usage()
			return
		}

		x509client, e := http3rd.BuildHttpClient(&params)
		if e != nil {
			logrus.Fatal(e)
		}
		logrus.Debug("Created HTTP x509client")

		req := &http3rd.MacaroonRequest{
			Resource:   args[0],
			Activities: args[1:],
			Lifetime:   macaroonLifetime,
		}
		m, e := http3rd.GetMacaroon(x509client, req)
		if e != nil {
			logrus.Fatal(e)
		}

		logrus.Info("Macaroon: ", m.Macaroon)
		logrus.Info("URL: ", m.Uri.TargetWithMacaroon)
	},
}

func init() {
	rootCmd.AddCommand(macaroonCmd)
	flags := macaroonCmd.Flags()
	flags.DurationVar(&macaroonLifetime, "lifetime", time.Minute, "Macaroon lifetime")
}
