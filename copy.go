package http3rd

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"gitlab.cern.ch/flutter/go-proxy"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type (
	Params struct {
		UserCert, UserKey string
		CAPath            string
		Insecure          bool
	}

	TokenRequest struct {
		Caveats []string `json:"caveats,omitempty"`
	}

	TokenResponse struct {
		Macaroon string `json:"macaroon"`
		Uri      struct {
			TargetWithMacaroon string `json:"targetWithMacaroon"`
			BaseWithMacaroon   string `json:"baseWithMacaroon"`
			Target             string `json:"target"`
			Base               string `json:"base"`
		} `json:"uri"`
	}
)

// getTokenFor returns a token for the resource
func getTokenFor(client *http.Client, resource string) (*TokenResponse, error) {
	payload := &TokenRequest{
		Caveats: []string{"activity:UPLOAD"},
	}
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method: "POST",
		Header: http.Header{},
	}
	req.Header.Add("Content-Type", "application/macaroon-request")
	req.URL, err = url.Parse(resource)
	if err != nil {
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(payloadData))
	req.ContentLength = int64(len(payloadData))

	reqRaw, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	logrus.Debug(string(reqRaw))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Response status code: ", resp.StatusCode)

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logrus.Debug("Response: ", string(respBody))

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	tokenResponse := &TokenResponse{}
	err = json.Unmarshal(respBody, tokenResponse)
	return tokenResponse, err
}

// http.Do only follows redirects for GET, HEAD, POST and PUT
// For COPY we have to do it ourselves (bummer)
func doWithRedirect(client *http.Client, r *http.Request) (resp *http.Response, err error) {
	jumps := 10

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
	}

	return
}

// requestRawCopy triggers the COPY method
func requestRawCopy(client *http.Client, source string, destination, macaroon string) error {
	var err error
	req := &http.Request{
		Method: "COPY",
		Header: http.Header{},
	}
	req.URL, err = url.Parse(source)
	if err != nil {
		return err
	}

	req.Header.Add("Destination", destination)
	req.Header.Add("X-No-Delegate", "true")
	req.Header.Add("TransferHeaderAuthorization", fmt.Sprint("BEARER ", macaroon))

	rawReq, err := httputil.DumpRequest(req, false)
	if err != nil {
		return err
	}
	logrus.Debug(string(rawReq))

	resp, err := doWithRedirect(client, req)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 != 2 {
		rawResp, _ := httputil.DumpResponse(resp, true)
		logrus.Debug(string(rawResp))
		return fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for line, err := reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
		logrus.Debug(line)
	}

	return nil
}

// DoHTTP3rdCopy triggers a third party copy
func DoHTTP3rdCopy(params *Params, source, destination string) error {
	logrus.Debug("User cert: ", params.UserCert)
	logrus.Debug("User key: ", params.UserKey)

	cert, err := tls.LoadX509KeyPair(params.UserCert, params.UserKey)
	if err != nil {
		return err
	}

	logrus.Debug("CA Path: ", params.CAPath)
	rootCerts, err := proxy.LoadCAPath(params.CAPath, false)
	if err != nil {
		return err
	}
	for _, ca := range rootCerts.CaByHash {
		logrus.Debug("CA: ", proxy.NameRepr(&ca.Subject))
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            rootCerts.CertPool,
			InsecureSkipVerify: params.Insecure,
		},
	}
	client := &http.Client{
		Transport: transport,
	}

	destinationToken, err := getTokenFor(client, destination)
	if err != nil {
		return err
	}

	logrus.Info("Got token ", destinationToken.Macaroon)

	return requestRawCopy(client, source, destination, destinationToken.Macaroon)
}
