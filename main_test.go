package main

import (
	"regexp"
	"testing"
)

func TestRegex(t *testing.T) {
	tests := []struct {
		name  string
		regex string
		cases []string
	}{
		{name: "CCTV1", regex: "^CCTV-?1([^0-9].*|$)", cases: []string{"CCTV1", "CCTV-1", "CCTV-1 HD", "CCTV-1 高清", "CCTV-11", "CCTV-11 高清"}},
		{name: "CGTN记录", regex: "^CGTN[ -]?(纪实|纪录|Documentary|DOCUMENTARY)$", cases: []string{"CGTN纪录"}},
	}
	for _, test := range tests {
		compile, err := regexp.Compile(test.regex)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range test.cases {
			t.Log(s, compile.MatchString(s))
		}
	}
}
