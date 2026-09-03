package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// spec 078 那張「類型 × 選擇」表逐格對。
func TestResolveEncounterChoice(t *testing.T) {
	for _, item := range []struct {
		note string
		in   gamepack.EncounterInputs
		want gamepack.EncounterOutcome
	}{
		{"類型 0 開打", gamepack.EncounterInputs{Kind: 0, Choice: gamepack.EncounterChoiceCombat},
			gamepack.EncounterOutcome{Store: true, ResultCode: 1}},
		{"類型 0 逃得掉", gamepack.EncounterInputs{Kind: 0, Choice: gamepack.EncounterChoiceFlee, SlowestMovement: 12, FleeThreshold: 9},
			gamepack.EncounterOutcome{Store: true, ResultCode: 2}},
		{"類型 0 逃不掉", gamepack.EncounterInputs{Kind: 0, Choice: gamepack.EncounterChoiceFlee, SlowestMovement: 6, FleeThreshold: 9},
			gamepack.EncounterOutcome{Store: true, ResultCode: 1}},
		{"類型 1 等待", gamepack.EncounterInputs{Kind: 1, Choice: gamepack.EncounterChoiceWait},
			gamepack.EncounterOutcome{Message: gamepack.EncounterMessageWait, Repeat: true}},
		// `24E9h`：PARLAY 距離大於零先拉近，距離為零才存 3。
		{"類型 1 交涉時拉近", gamepack.EncounterInputs{Kind: 1, Choice: gamepack.EncounterChoiceParley, Distance: 2},
			gamepack.EncounterOutcome{Approach: true, Repeat: true}},
		{"類型 1 貼身交涉成立", gamepack.EncounterInputs{Kind: 1, Choice: gamepack.EncounterChoiceParley, Distance: 0},
			gamepack.EncounterOutcome{Store: true, ResultCode: 3}},
		// `2483h`：ADVANCE 距離為零只印「雙方按兵不動」再問一次。
		{"類型 1 逼近到底", gamepack.EncounterInputs{Kind: 1, Choice: gamepack.EncounterChoiceAdvance, Distance: 0},
			gamepack.EncounterOutcome{Message: gamepack.EncounterMessageWait, Repeat: true}},
		{"類型 2 追得上", gamepack.EncounterInputs{Kind: 2, Choice: gamepack.EncounterChoiceCombat, FastestMovement: 12, AdvanceThreshold: 9},
			gamepack.EncounterOutcome{Store: true, ResultCode: 1}},
		{"類型 2 追不上", gamepack.EncounterInputs{Kind: 2, Choice: gamepack.EncounterChoiceCombat, FastestMovement: 6, AdvanceThreshold: 9},
			gamepack.EncounterOutcome{Store: true, ResultCode: 0, Message: gamepack.EncounterMessageFlee}},
		{"類型 3 開打", gamepack.EncounterInputs{Kind: 3, Choice: gamepack.EncounterChoiceCombat},
			gamepack.EncounterOutcome{Store: true, ResultCode: 1}},
		{"類型 3 逃跑不用擲", gamepack.EncounterInputs{Kind: 3, Choice: gamepack.EncounterChoiceFlee},
			gamepack.EncounterOutcome{Store: true, ResultCode: 2}},
		// `260Ch`：類型 3 的等待與 ADVANCE 走同一支。
		{"類型 3 逼近到底", gamepack.EncounterInputs{Kind: 3, Choice: gamepack.EncounterChoiceAdvance, Distance: 0},
			gamepack.EncounterOutcome{Message: gamepack.EncounterMessageWait, Repeat: true}},
		// `2686h`：類型 3 的 PARLAY 貼身就存 3。索寇要塞登陸的那個亡魂
		// （ecl4/21 `AA0Ah`，五格類型表是 0 3 1 1 4）走的就是類型 4 這一支。
		{"類型 3 貼身交涉成立", gamepack.EncounterInputs{Kind: 3, Choice: gamepack.EncounterChoiceParley, Distance: 0},
			gamepack.EncounterOutcome{Store: true, ResultCode: 3}},
		// `26F4h`：類型 4 的等待、ADVANCE、PARLAY 三個走同一支。
		{"類型 4 貼身交涉成立", gamepack.EncounterInputs{Kind: 4, Choice: gamepack.EncounterChoiceParley, Distance: 0},
			gamepack.EncounterOutcome{Store: true, ResultCode: 3}},
		{"類型 4 距離還在就拉近", gamepack.EncounterInputs{Kind: 4, Choice: gamepack.EncounterChoiceWait, Distance: 3},
			gamepack.EncounterOutcome{Approach: true, Repeat: true}},
		{"類型 4 開打", gamepack.EncounterInputs{Kind: 4, Choice: gamepack.EncounterChoiceCombat},
			gamepack.EncounterOutcome{Store: true, ResultCode: 1}},
	} {
		got, err := gamepack.ResolveEncounterChoice(item.in)
		if err != nil {
			t.Fatalf("%s: %v", item.note, err)
		}
		if got != item.want {
			t.Fatalf("%s: got %+v, want %+v", item.note, got, item.want)
		}
	}
}

// 沒讀出來的組合要回錯誤，不要猜一個看起來合理的行為。
func TestResolveEncounterChoiceRefusesUnreadCombinations(t *testing.T) {
	for _, in := range []gamepack.EncounterInputs{
		{Kind: 5, Choice: gamepack.EncounterChoiceCombat},
		{Kind: 9, Choice: gamepack.EncounterChoiceParley},
	} {
		if _, err := gamepack.ResolveEncounterChoice(in); err == nil {
			t.Fatalf("kind %d choice %d was resolved without evidence", in.Kind, in.Choice)
		}
	}
}

// 沒有地圖上的怪物群時第四項是 PARLAY，而且它指到類型表的第 4 格而不是第 5 格。
func TestEncounterMenuVariant(t *testing.T) {
	parley := gamepack.EncounterMenuOptions(2, false)
	if parley[3] != "PARLAY" {
		t.Fatalf("fourth option is %q, want PARLAY", parley[3])
	}
	if index := gamepack.EncounterChoiceIndex(3, parley); index != gamepack.EncounterChoiceParley {
		t.Fatalf("PARLAY maps to table slot %d, want %d", index, gamepack.EncounterChoiceParley)
	}
	advance := gamepack.EncounterMenuOptions(2, true)
	if advance[3] != "ADVANCE" {
		t.Fatalf("fourth option is %q, want ADVANCE", advance[3])
	}
	if index := gamepack.EncounterChoiceIndex(3, advance); index != gamepack.EncounterChoiceAdvance {
		t.Fatalf("ADVANCE maps to table slot %d, want %d", index, gamepack.EncounterChoiceAdvance)
	}
}
