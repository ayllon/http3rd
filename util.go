package http3rd

import (
	"crypto/tls"
	"errors"
	"github.com/sirupsen/logrus"
	"gitlab.cern.ch/flutter/go-proxy"
	"io"
	"net/http"
	"net/url"
	"io/ioutil"
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

// http.Do only follows redirects for GET, HEAD, POST and PUT
// For COPY we have to do it ourselves (bummer)
func DoWithRedirect(client *http.Client, r *http.Request) (resp *http.Response, err error) {
	jumps := 10

	// Wrap the body to avoid it being close on a redirect
	originalBody := r.Body
	if originalBody != nil {
		defer originalBody.Close()
		r.Body = ioutil.NopCloser(originalBody)
	}

	for {
		resp, err = client.Do(r)
		if err != nil || resp.StatusCode/100 != 3 {
			return
		}
		if jumps--; jumps <= 0 {
			err = errors.New("stopped after 10 redirects")
			return
		}
		location := resp.Header.Get("Location")
		r.URL, err = url.Parse(location)
		if err != nil {
			return
		}
		logrus.Debug("Following redirect: ", location)
		if seeker, ok := originalBody.(io.Seeker); ok {
			logrus.Debug("Rewind file")
			_, err := seeker.Seek(0, io.SeekStart)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}
