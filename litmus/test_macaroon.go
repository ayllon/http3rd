package main

import (
	"encoding/base64"
	"fmt"
	"github.com/ayllon/http3rd"
	"github.com/go-macaroon/macaroon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/studio-b12/gowebdav"
	"gopkg.in/check.v1"
	"net/http"
	"net/url"
	"path"
	"time"
)

var (
	baseURL = ""
	filter  = ""
)

// DecodeMacaroon returns a macaroon.M from the Base64 representation pased as a parameter
func DecodeMacaroon(encoded string) (*macaroon.Macaroon, error) {
	decoded := make([]byte, base64.RawURLEncoding.DecodedLen(len(encoded)))
	_, e := base64.RawURLEncoding.Decode(decoded, []byte(encoded))
	if e != nil {
		return nil, fmt.Errorf("Could not base64-decode: %s", e)
	}
	M := &macaroon.Macaroon{}
	e = M.UnmarshalBinary(decoded)
	return M, e
}

// EncodeMacaroon returns a serialized base64 macaroon
func EncodeMacaroon(M *macaroon.Macaroon) (string, error) {
	token, e := M.MarshalBinary()
	if e != nil {
		return "", e
	}
	token64 := make([]byte, base64.RawURLEncoding.EncodedLen(len(token)))
	base64.RawURLEncoding.Encode(token64, token)
	return string(token64), e
}

// TryDownload tries to download a file (without actually doing so)
// Returns the HTTP status code, or an error if can't even try
func TryDownload(client *http.Client, uri, token string) (int, error) {
	getReq := &http.Request{
		Method: "GET",
		Header: make(http.Header),
	}
	getReq.Header.Add("Authorization", "BEARER "+token)
	getReq.URL, _ = url.Parse(uri)

	resp, err := client.Do(getReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// Macaroon test suite
type MacaroonTestSuite struct {
	// It has X509 client credentials setup. Used to obtain macaroons, and test setup.
	x509client *http.Client
	// Does not have x509 credentials setup, so things can only work if the token is proper.
	client *http.Client
	// Base path
	path string
	// Any file we can use to test downloads and such (can't download a directory!)
	file    string
	fileURL string
}

// NewDavClient creates a new initialized DAV client
func (s *MacaroonTestSuite) NewDavClient(token string) *gowebdav.Client {
	dav := gowebdav.NewClient(baseURL, "", "")
	dav.SetTransport(s.client.Transport)
	if token != "" {
		dav.SetHeader("Authorization", "BEARER "+token)
	}
	return dav
}

// SetUpSuite looks for a file that can be used to test the downloading
func (s *MacaroonTestSuite) SetUpSuite(c *check.C) {
	dav := gowebdav.NewClient(baseURL, "", "")
	dav.SetTransport(s.x509client.Transport)
	files, e := dav.ReadDir("/")
	if e != nil {
		c.Fatal(e)
	}
	for _, f := range files {
		if !f.IsDir() {
			s.file = f.Name()
			logrus.Infof("Using %s for file tests", s.file)
			break
		}
	}
	if s.file == "" {
		c.Fatal("Could not find a file to use")
	}

	p, _ := url.Parse(baseURL)
	s.path = p.Path

	if baseURL[len(baseURL)-1] != '/' {
		s.fileURL = baseURL + "/" + s.file
	} else {
		s.fileURL = baseURL + s.file
	}
}

// TestPlainRequest asks for a Macaroon and checks it can be deserialized, that's all
func (s *MacaroonTestSuite) TestPlainRequest(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	resp, err := http3rd.GetMacaroon(s.x509client, req)
	if err != nil {
		c.Fatal(err)
	}

	M, e := DecodeMacaroon(resp.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	c.Log("Decoded with ID ", M.Id())

	for _, caveat := range M.Caveats() {
		c.Log("Caveat: ", caveat.Id)
	}
}

// TestNoCertRequest asks for a Macaroon using the client without certificates
func (s *MacaroonTestSuite) TestNoCertRequest(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	_, e := http3rd.GetMacaroon(s.client, req)
	if e == nil {
		c.Error("The request should have failed")
	}
}

// TestAccessNoMacaroon tries to stat without a token
func (s *MacaroonTestSuite) TestAccessNoMacaroon(c *check.C) {
	dav := s.NewDavClient("")
	_, e := dav.ReadDir("/")
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestAccess asks for a token, and does a listing, which should work
func (s *MacaroonTestSuite) TestAccess(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(m.Macaroon)
	_, e = dav.Stat("/")
	if e != nil {
		c.Error(e)
	}
}

// TestRandomGarbage uses some random garbage as token
func (s *MacaroonTestSuite) TestRandomGarbage(c *check.C) {
	dav := s.NewDavClient("Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9Hixkk")
	_, e := dav.ReadDir("/")
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestMadeupMacaroon builds a made up macaroon and try to use it
func (s *MacaroonTestSuite) TestMadeupMacaroon(c *check.C) {
	M, e := macaroon.New([]byte("1234"), "abcde", s.path)
	if e != nil {
		c.Fatal(e)
	}
	M.AddFirstPartyCaveat("id:2002;2002;paul")
	M.AddFirstPartyCaveat("dn:/DC=ch/DC=cern/OU=Organic Units/OU=Users/CN=ftssuite/CN=737188/CN=Robot: fts3 testsuite")
	M.AddFirstPartyCaveat("before:2100-05-01T12:00:00.000Z")
	M.AddFirstPartyCaveat("activity:LIST")
	M.AddFirstPartyCaveat("path:" + s.path)

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(string(token))
	_, e = dav.ReadDir("/")
	if e == nil {
		c.Fatal("Expecting an error")
	}
}

// TestBadResource asks for a Macaroon, and uses it for a different URL
func (s *MacaroonTestSuite) TestBadResource(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	// Download is not fine
	code, e := TryDownload(s.client, s.fileURL, m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}
	if code != 403 {
		c.Error("Expecting a 403, got ", code)
	}
}

// TestAskBogusCaveat tries to fool the remote endpoint trying to override a caveat
func (s *MacaroonTestSuite) TestAskBogusCaveat(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	M, e := DecodeMacaroon(m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	M.AddFirstPartyCaveat("path:" + path.Join(s.path, s.file))

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(string(token))
	_, e = dav.Stat("/" + s.file)
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestExpired asks for a token, sleeps a bit, trys to perform the action
func (s *MacaroonTestSuite) TestExpired(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   2 * time.Second,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	// First attempt on time
	dav := s.NewDavClient(m.Macaroon)
	_, e = dav.Stat("/")
	if e != nil {
		c.Fatal(e)
	}

	// Sleep and try again
	time.Sleep(3 * time.Second)
	_, e = dav.ReadDir("/")
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestExpiredReduce asks for a long lived token, and reduce its lifetime
// Reducing lifetime is acceptable
func (s *MacaroonTestSuite) TestExpiredReduce(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	M, e := DecodeMacaroon(m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	before := time.Now().Add(time.Second).UTC()
	M.AddFirstPartyCaveat(fmt.Sprint("before:", before.Format(time.RFC3339)))

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	time.Sleep(2 * time.Second)

	dav := s.NewDavClient(token)
	_, e = dav.ReadDir("/")
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestExpiredIncrease tries to increase the token lifetime, which isn't acceptable
func (s *MacaroonTestSuite) TestExpiredIncrease(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   baseURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Second,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	M, e := DecodeMacaroon(m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	before := time.Now().Add(time.Hour).UTC()
	M.AddFirstPartyCaveat(fmt.Sprint("before:", before.Format(time.RFC3339)))

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	time.Sleep(2 * time.Second)

	dav := s.NewDavClient(token)
	_, e = dav.ReadDir("/")
	if e == nil {
		c.Error("Expecting an error")
	}
}

// TestInvalidActivity asks for a given activity (LIST) and tries something else (DOWNLOAD)
func (s *MacaroonTestSuite) TestInvalidActivity(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   s.fileURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(m.Macaroon)

	// Stat (LIST) is fine
	_, e = dav.Stat("/" + s.file)
	if e != nil {
		c.Fatal(e)
	}

	// Download is not fine
	code, e := TryDownload(s.client, s.fileURL, m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}
	if code != 403 {
		c.Error("Expecting a 403, got ", code)
	}
}

// TestReduceActivity is similar to TestInvalidActivity, but this time it is the client
// who reduces the activities
func (s *MacaroonTestSuite) TestReduceActivity(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   s.fileURL,
		Activities: []string{http3rd.List, http3rd.Download},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	M, e := DecodeMacaroon(m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	M.AddFirstPartyCaveat("activity:LIST")

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(token)

	// Stat (LIST) is fine
	_, e = dav.Stat("/" + s.file)
	if e != nil {
		c.Fatal(e)
	}

	// Download is not fine
	code, e := TryDownload(s.client, s.fileURL, token)
	if e != nil {
		c.Fatal(e)
	}
	if code != 403 {
		c.Error("Expecting a 403, got ", code)
	}
}

// TestIncreaseActivity tries to fool the server adding an activity
func (s *MacaroonTestSuite) TestIncreaseActivity(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   s.fileURL,
		Activities: []string{http3rd.List},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	M, e := DecodeMacaroon(m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}

	M.AddFirstPartyCaveat("activity:LIST,DOWNLOAD")

	token, e := EncodeMacaroon(M)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(token)

	// Stat (LIST) is fine
	_, e = dav.Stat("/" + s.file)
	if e != nil {
		c.Fatal(e)
	}

	// Download is not fine
	code, e := TryDownload(s.client, s.fileURL, m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}
	if code != 403 {
		c.Error("Expecting a 403, got ", code)
	}
}

// TestDownloadAndStat asks for List and Download, both must work
func (s *MacaroonTestSuite) TestDownloadAndStat(c *check.C) {
	req := &http3rd.MacaroonRequest{
		Resource:   s.fileURL,
		Activities: []string{http3rd.List, http3rd.Download},
		Lifetime:   time.Minute,
	}
	m, e := http3rd.GetMacaroon(s.x509client, req)
	if e != nil {
		c.Fatal(e)
	}

	dav := s.NewDavClient(m.Macaroon)

	// Stat (LIST) is fine
	_, e = dav.Stat("/" + s.file)
	if e != nil {
		c.Fatal(e)
	}

	// Download is not fine
	code, e := TryDownload(s.client, s.fileURL, m.Macaroon)
	if e != nil {
		c.Fatal(e)
	}
	if code != 200 {
		c.Error("Expecting a 200, got ", code)
	}
}

// Run the macaroon test suite
var macaroonTestCmd = &cobra.Command{
	Use: "macaroon",
	Run: func(cmd *cobra.Command, args []string) {
		x509client, e := http3rd.BuildHttpClient(&params)
		if e != nil {
			logrus.Fatal(e)
		}
		logrus.Debug("Created HTTP x509client")

		client, e := http3rd.BuildHttpClient(&http3rd.Params{
			Insecure: params.Insecure,
			CAPath:   params.CAPath,
		})
		if e != nil {
			logrus.Fatal(e)
		}

		msuite := &MacaroonTestSuite{
			x509client: x509client,
			client:     client,
			file:       "",
		}

		suite := check.Suite(msuite)
		conf := &check.RunConf{
			Verbose: true,
			Filter:  filter,
		}
		result := check.Run(suite, conf)

		logrus.Info("Success: ", result.Succeeded)
		logrus.Info("Skipped: ", result.Skipped)
		if result.Failed > 0 {
			logrus.Warn("Failed: ", result.Failed)
		}
		if result.Missed > 0 {
			logrus.Warn("Missed: ", result.Missed)
		}
		if result.Panicked > 0 {
			logrus.Error("Panicked: ", result.Panicked)
		}
	},
}

func init() {
	testCmd.AddCommand(macaroonTestCmd)
	flags := macaroonTestCmd.Flags()
	flags.StringVar(&baseURL, "url", "https://arioch.cern.ch/dpm/cern.ch/home/dteam/", "Base URL for the tests")
	flags.StringVar(&filter, "filter", "", "Filter tests")
}
