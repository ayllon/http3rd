package http3rd

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
	"errors"
)

type (
	// Params configures the third party copy request
	Params struct {
		UserCert, UserKey string
		CAPath            string
		Insecure          bool
	}
)

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

// buildCopyRequest returns an initialized HTTP COPY request
func buildCopyRequest(source, destination, macaroon string) (*http.Request, error) {
	var err error

	req := &http.Request{
		Method: "COPY",
		Header: http.Header{},
	}
	req.URL, err = url.Parse(source)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Destination", destination)
	req.Header.Add("X-No-Delegate", "true")
	req.Header.Add("Credential", "none")
	req.Header.Add("TransferHeaderAuthorization", fmt.Sprint("BEARER ", macaroon))
	return req, nil
}

// requestRawCopy triggers the COPY method
func requestRawCopy(client *http.Client, source string, destination, macaroon string) error {
	req, err := buildCopyRequest(source, destination, macaroon)
	if err != nil {
		return err
	}

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
func DoHTTP3rdCopy(params *Params, lifetime time.Duration, source, destination string) error {
	client, err := BuildHttpClient(params)
	if err != nil {
		return err
	}

	destinationToken, err := GetMacaroon(client, &MacaroonRequest{
		Resource:   destination,
		Lifetime:   lifetime,
		Activities: []string{Upload},
	})
	if err != nil {
		return err
	}

	logrus.Info("Got macaroon ", destinationToken.Macaroon)

	// TODO: Parse response
	return requestRawCopy(client, source, destination, destinationToken.Macaroon)
}
