package main

import "testing"

func TestParseTargetsNormalizesAndRejectsBadInput(t *testing.T) {
	targets, err := parseTargets("0x4AC1, 4ab1,4AC1")
	if err != nil || len(targets) != 2 || !targets[0x4AC1] || !targets[0x4AB1] {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
	if _, err := parseTargets("not-hex"); err == nil {
		t.Fatal("invalid target accepted")
	}
}
