package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/ayllon/http3rd"
	"gitlab.cern.ch/flutter/go-proxy"
	"time"
)

// Setup the logger
func setupLogger(verbose bool) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

// Return the user certificate and private key to use
// If flagCert is not set, it will try to figure it out
func setupUserCredentials(flagCert, flagKey string) (string, string) {
	if flagCert != "" {
		if flagKey == "" {
			return flagCert, flagCert
		}
		return flagCert, flagKey
	}
	cert, key, err := proxy.GetCertAndKeyLocation()
	if err != nil {
		logrus.Fatal(err)
	}
	return cert, key
}

// Entry point
func main() {
	debugFlag := flag.Bool("debug", false, "Enable debug output")
	certFlag := flag.String("cert", "", "User certificate")
	keyFlag := flag.String("key", "", "User private key")
	capathFlag := flag.String("capath", "/etc/grid-security/certificates", "CA Path")
	insecureFlag := flag.Bool("insecure", false, "Do not validate the remote certificates")
	onlyMacaroon := flag.Bool("macaroon", false, "Only request and print the macaroon for the given url")
	lifetime := flag.Duration("lifetime", time.Minute, "Lifetime of the requested token")

	flag.Parse()
	if *onlyMacaroon && flag.NArg() != 1 {
		logrus.Fatal("Exactly one argument required for --macaroon")
	} else if !*onlyMacaroon && flag.NArg() != 2 {
		logrus.Fatal("Exactly two arguments required: source destination")
	}

	setupLogger(*debugFlag)
	ucert, ukey := setupUserCredentials(*certFlag, *keyFlag)

	source := flag.Arg(0)

	params := &http3rd.Params{
		UserCert: ucert,
		UserKey:  ukey,
		CAPath:   *capathFlag,
		Insecure: *insecureFlag,
		Lifetime: *lifetime,
	}

	if *onlyMacaroon {
		macaroon, err := http3rd.GetMacaroon(params, source)
		if err != nil {
			logrus.Fatal(err)
		}
		fmt.Println("Macaroon:", macaroon.Macaroon)
		fmt.Println("URL + Macaroon:", macaroon.Uri.TargetWithMacaroon)
	} else {
		destination := flag.Arg(1)
		err := http3rd.DoHTTP3rdCopy(params, source, destination)
		if err != nil {
			logrus.Fatal(err)
		}
	}
}
