package main

import (
	"github.com/ayllon/http3rd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	copyLifetime = 5 * time.Minute
)

var copyCmd = &cobra.Command{
	Use: "copy <src> <dst>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			logrus.Fatal("Expecting two arguments")
		}
		e := http3rd.DoHTTP3rdCopy(&params, copyLifetime, args[0], args[1])
		if e != nil {
			logrus.Fatal(e)
		}
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	flags := copyCmd.Flags()
	flags.DurationVar(&copyLifetime, "lifetime", 5*time.Minute, "Duration of the bearer token")
}
