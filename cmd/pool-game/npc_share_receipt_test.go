package main

// NPC 分錢頁的版面對原版收據（`docs/audit/dosgolem-npc-share-screen.json`，spec 150，issue #111）：
// 原版那一份是從 `slums.state` 出發、把第二與第四個人的 `+84h`／`+85h` 改成 B2h／1 的診斷樣本，
// 兩行落在第 5 欄的第 5 與第 7 列。這裡用同一份隊伍（B..F）走同一段交件，從 `Update()` 送鍵。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestNPCSharePageMatchesTheOriginalLayout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-npc-share-screen.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Generator string `json:"generator"`
		Shots     []struct {
			Why   string `json:"why"`
			Lines []struct {
				Row, Column int
				Text        string
			} `json:"hides_lines"`
		} `json:"shots"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Generator != "dosgolem" || len(receipt.Shots) == 0 || receipt.Shots[0].Why != "npc-share" ||
		len(receipt.Shots[0].Lines) != 2 {
		t.Fatalf("receipt %+v", receipt)
	}
	application, session, _ := slumsCommissionApp(t, 25)
	party := receiptParty()
	application.state.CharacterLibrary = party
	for _, slot := range []int{1, 3} {
		record := make([]byte, 0x11D) // 285-byte 角色記錄（overlay-07 `1B6Dh` 的 GetMem(11Dh)）
		record[gamepack.MoraleOffset], record[gamepack.MoraleOffset+1] = 0xB2, 0x01
		party[slot].NPC, party[slot].Record = true, record
	}
	application.state.Party = party
	handInAtCityHall(t, application, session)
	if application.treasureStage != treasureNPCShare {
		t.Fatalf("stage %d, want the NPC share page", application.treasureStage)
	}
	lines := application.postCombatPageLines()
	if len(lines) != len(receipt.Shots[0].Lines) {
		t.Fatalf("remake %+v, original %+v", lines, receipt.Shots[0].Lines)
	}
	for index, want := range receipt.Shots[0].Lines {
		got := lines[index]
		if got.row != want.Row || got.column != want.Column || strings.ToUpper(got.text) != want.Text {
			t.Fatalf("line %d: remake (%d,%d) %q, original (%d,%d) %q", index,
				got.column, got.row, got.text, want.Column, want.Row, want.Text)
		}
	}
}
