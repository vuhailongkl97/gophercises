package main

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig("./config.yaml")

	expectedRet := config{"123", "123", "http://localhost:18080/threshold/",
		"http://localhost:18080/disable/0", "http://localhost:18080/disable/1",
		20, "http://localhost:1234"}

	if err != nil {
		t.Fatal(err)
	} else if cfg != expectedRet {
		fmt.Printf("%v\n%v\n", cfg, expectedRet)
	}

}
