package http3rd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	Download = "DOWNLOAD"
	Upload   = "UPLOAD"
	List     = "LIST"
	Delete   = "DELETE"
	Manage   = "MANAGE"
)

type (
	// jsonMacaroonRequest models a Macaroon request sent to the server
	jsonMacaroonRequest struct {
		// List of serialized caveats
		Caveats []string `json:"caveats,omitempty"`
	}

	// MacaroonRequest wraps the supported request fields for a Grid SE Macaroon
	MacaroonRequest struct {
		Resource   string
		Lifetime   time.Duration
		Activities []string
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

// buildHTTPRequest builds a Macaroon request
func buildHTTPRequest(request *MacaroonRequest) (*http.Request, error) {
	payload := &jsonMacaroonRequest{
		Caveats: []string{
			fmt.Sprint("activity:", strings.Join(request.Activities, ",")),
		},
	}

	if request.Lifetime > 0 {
		before := time.Now().Add(request.Lifetime).UTC()
		payload.Caveats = append(payload.Caveats, fmt.Sprint("before:", before.Format(time.RFC3339)))
	}

	payloadData, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}

	req := &http.Request{
		Method: "POST",
		Header: http.Header{},
	}
	req.Header.Add("Content-Type", "application/macaroon-request")
	req.URL, e = url.Parse(request.Resource)
	if e != nil {
		return nil, e
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(payloadData))
	req.ContentLength = int64(len(payloadData))
	return req, nil
}

// GetMacaroon returns a token for the resource
func GetMacaroon(client *http.Client, request *MacaroonRequest) (*MacaroonResponse, error) {
	req, e := buildHTTPRequest(request)
	if e != nil {
		return nil, e
	}

	reqRaw, e := httputil.DumpRequest(req, true)
	if e != nil {
		return nil, e
	}
	logrus.Debug(string(reqRaw))

	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	logrus.Debug("Response status code: ", resp.StatusCode)

	respBody, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return nil, e
	}

	logrus.Debug("Response: ", string(respBody))

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	tokenResponse := &MacaroonResponse{}
	e = json.Unmarshal(respBody, tokenResponse)
	return tokenResponse, e
}
