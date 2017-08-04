package http3rd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type (
	// MacaroonRequest models a Macaroon request sent to the server
	MacaroonRequest struct {
		// List of caveats
		Caveats []string `json:"caveats,omitempty"`
	}

	// MacaroonResponse models the reply from the server
	MacaroonResponse struct {
		Macaroon string `json:"macaroon"`
		Uri      struct {
			TargetWithMacaroon string `json:"targetWithMacaroon"`
			BaseWithMacaroon   string `json:"baseWithMacaroon"`
			Target             string `json:"target"`
			Base               string `json:"base"`
		} `json:"uri"`
	}
)

// buildMacaroonRequest builds a Macaroon request
func buildMacaroonRequest(lifetime time.Duration, resource string) (*http.Request, error) {
	before := time.Now().Add(lifetime).UTC()

	payload := &MacaroonRequest{
		Caveats: []string{
			"activity:UPLOAD",
			fmt.Sprint("before:", before.Format(time.RFC3339)),
		},
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
	return req, nil
}

// getMacaroon returns a token for the resource
func getMacaroon(client *http.Client, lifetime time.Duration, resource string) (*MacaroonResponse, error) {
	req, err := buildMacaroonRequest(lifetime, resource)
	if err != nil {
		return nil, err
	}

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

	tokenResponse := &MacaroonResponse{}
	err = json.Unmarshal(respBody, tokenResponse)
	return tokenResponse, err
}
