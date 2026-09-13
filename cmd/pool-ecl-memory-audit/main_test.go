package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTargetsNormalizesAndRejectsBadInput(t *testing.T) {
	targets, err := parseTargets("0x4AC1, 4ab1,4AC1")
	if err != nil || len(targets) != 2 || !targets[0x4AC1] || !targets[0x4AB1] {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
	if _, err := parseTargets("not-hex"); err == nil {
		t.Fatal("invalid target accepted")
	}
}

// 簽入的清冊要保留 spec 038 已證實的三類 producer；只測參數 parser 會讓
// 真正的研究證據被重生流程悄悄刪掉也仍然全綠。
func TestCheckedInAuditPinsTheGraveyardProducers(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dos-ecl-campaign-memory-refs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, item := range got.References {
		counts[item.Target+":"+item.Name]++
	}
	if counts["0x4AC1:ADD"] < 10 {
		t.Errorf("4AC1h 只有 %d 個 ADD producer，至少應有 10 個", counts["0x4AC1:ADD"])
	}
	if counts["0x4A96:SAVE"] == 0 {
		t.Error("4A96h 的 SAVE producer 不見了")
	}
	if counts["0x4AB1:SAVE"] == 0 {
		t.Error("4AB1h 的 SAVE producer 不見了")
	}
}
