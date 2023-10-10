package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		//Test request Parameters
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})
	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"

	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}

}

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("wrong length random string returned")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		//setup a pipe to avoid buffer
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer writer.Close()
			defer wg.Done()
			//
			part, err := writer.CreateFormFile("file", "./testData/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testData/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()
			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())
		var testTools Tools
		testTools.AllowedFileType = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(request, "./testData/uploades/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testData/uploades/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exists: %s", e.name, err.Error())
			}
			_ = os.Remove(fmt.Sprintf("./testData/uploades/%s", uploadedFiles[0].NewFileName))
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none receieved", e.name)
		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()
		//
		part, err := writer.CreateFormFile("file", "./testData/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testData/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}

	}()
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())
	var testTools Tools
	uploadFiles, err := testTools.UploadOneFile(request, "./testData/uploades/", true)
	if err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(fmt.Sprintf("./testData/uploades/%s", uploadFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist:%s", err.Error())
	}
	_ = os.Remove(fmt.Sprintf("./testData/uploades/%s", uploadFiles.NewFileName))
}

func TestTools_CreateDirNotExist(t *testing.T) {
	var testTools Tools
	err := testTools.CreateDirIfNotExists("./testData/myDir")
	if err != nil {
		t.Error(err)
	}
	err = testTools.CreateDirIfNotExists("./testData/myDir")
	if err != nil {
		t.Error(err)
	}
	_ = os.Remove("./testData/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all GOOD men! + fish & such &^123", expected: "now-is-the-time-for-all-good-men-fish-such-123", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools
	for _, e := range slugTests {
		slug, err := testTools.Sligify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error receieved when none expected: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	var testTools Tools
	testTools.DownloadStaticFile(rr, req, "./testData", "file.PNG", "down.PNG")
	res := rr.Result()
	defer res.Body.Close()
	if k := res.Header["Content-Length"][0]; k != "323008" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}
	if res.Header["Content-Disposition"][0] != "attachment; filename=\"down.PNG\"" {
		t.Error("wrong content disposition")
	}
	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "valid json", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo":1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo":"1"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax field in json", json: `{"fooo":"1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown fields in json", json: `{"fooo":"1"`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"jack":"rayan"}`, errorExpected: false, maxSize: 10, allowUnknown: true},
	{name: "not a json", json: `jack rayan`, errorExpected: true, maxSize: 10, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools
	for _, e := range jsonTests {
		testTools.MaxJSONSize = e.maxSize
		testTools.AllowUnknownFields = e.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}
		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none receieved", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: err not expected,but one receieved: %s", e.name, err.Error())

		}
		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("FOO", "BAR")
	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to erite JSON: %v", err)
	}

}
func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}
	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("receieved error when decoding JSON", err)
	}
	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}
