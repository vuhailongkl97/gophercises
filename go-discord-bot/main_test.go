package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig("./config.yaml")

	expectedRet := config{"123", "123", "http://localhost:18080/threshold/",
		"http://localhost:18080/disable/0", "http://localhost:18080/disable/1",
		20, "http://localhost:1234"}

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cfg, expectedRet)
}

type MockNotifingAdapter struct{}

func (tee *MockNotifingAdapter) ChannelFileSendWithMessage(cfg config, fileName string, rdr io.Reader) error {
	fmt.Println("call ChannelFileSendWithMessage")
	return nil
}
func (tee *MockNotifingAdapter) Open() error {
	fmt.Println("Call Open")
	return nil
}

func (tee *MockNotifingAdapter) Close() error {
	return nil
}

func TestServeHTTP(t *testing.T) {

	notifierInterface = &MockNotifingAdapter{}
	rdr := strings.NewReader("wallpapers.jpg")
	req := httptest.NewRequest(http.MethodPost, "/updated", rdr)

	w := httptest.NewRecorder()

	counter = 1
	restTime = time.Now().Add(-2 * time.Hour)
	serveHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	content, err := ioutil.ReadAll(result.Body)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ok", string(content))
	assert.Equal(t, default_counter, counter)
	assert.Equal(t, restTime.After(time.Now().Add(time.Duration(cfg.RestTime-1)*time.Minute)), true)
}

func TestServeHTTP2(t *testing.T) {

	notifierInterface = &MockNotifingAdapter{}
	rdr := strings.NewReader("wallpapers.jpg")
	req := httptest.NewRequest(http.MethodPost, "/updated", rdr)

	w := httptest.NewRecorder()

	counter = 2
	restTime = time.Now().Add(2 * time.Hour)
	serveHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	content, err := ioutil.ReadAll(result.Body)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ok", string(content))
	assert.Equal(t, 1, counter)
}

func TestServeHTTPmax(t *testing.T) {

	notifierInterface = &MockNotifingAdapter{}

	counter = default_counter
	restTime = time.Now().Add(2 * time.Hour)
	expect_counters := []int{5, 4, 3, 2, 1}
	for _, v := range expect_counters {
		rdr := strings.NewReader("wallpapers.jpg")
		req := httptest.NewRequest(http.MethodPost, "/updated", rdr)
		w := httptest.NewRecorder()
		serveHTTP(w, req)
		result := w.Result()
		defer result.Body.Close()
		content, err := ioutil.ReadAll(result.Body)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "ok", string(content))
		assert.Equal(t, v, counter)
	}

}

func TestInitHardware(t *testing.T) {

}

func TestReadStatus(t *testing.T) {
	str := readStatus()
	fmt.Println(str)
}
