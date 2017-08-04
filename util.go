package http3rd

import (
	"crypto/tls"
	"github.com/sirupsen/logrus"
	"gitlab.cern.ch/flutter/go-proxy"
	"net/http"
)

func BuildHttpTransport(params *Params) (*http.Transport, error) {
	logrus.Debug("User cert: ", params.UserCert)
	logrus.Debug("User key: ", params.UserKey)

	certificates := []tls.Certificate{}

	if params.UserCert != "" {
		cert, e := tls.LoadX509KeyPair(params.UserCert, params.UserKey)
		if e != nil {
			return nil, e
		}
		certificates = append(certificates, cert)
	}

	logrus.Debug("CA Path: ", params.CAPath)
	rootCerts, e := proxy.LoadCAPath(params.CAPath, false)
	if e != nil {
		return nil, e
	}
	for _, ca := range rootCerts.CaByHash {
		logrus.Debug("CA: ", proxy.NameRepr(&ca.Subject))
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       certificates,
			RootCAs:            rootCerts.CertPool,
			InsecureSkipVerify: params.Insecure,
		},
	}, nil
}

// BuildHttpClient returns an initialized http.Client
func BuildHttpClient(params *Params) (*http.Client, error) {
	transport, e := BuildHttpTransport(params)
	if e != nil {
		return nil, e
	}
	return &http.Client{
		Transport: transport,
	}, nil
}
