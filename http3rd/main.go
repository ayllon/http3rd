package main 

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/ayllon/http3rd"
	"gitlab.cern.ch/flutter/go-proxy"
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

	flag.Parse()
	if flag.NArg() != 2 {
		logrus.Fatal("Exactly two arguments required: source destination")
	}

	setupLogger(*debugFlag)
	ucert, ukey := setupUserCredentials(*certFlag, *keyFlag)
	source := flag.Arg(0)
	destination := flag.Arg(1)

	err := http3rd.DoHTTP3rdCopy(&http3rd.Params{
		UserCert: ucert,
		UserKey:  ukey,
		CAPath:   *capathFlag,
		Insecure: *insecureFlag,
	}, source, destination)
	if err != nil {
		logrus.Fatal(err)
	}
}
