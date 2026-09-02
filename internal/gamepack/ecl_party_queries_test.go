package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// `22h` 認的是職業碼 4 與 0Ah。《光芒之池》的建角表沒有這兩個碼，
// 所以自己建的隊伍一律得到 0——那是正確結果，不是漏接。
func TestPartySurpriseAlertOnlyFiresForTheTwoClassCodes(t *testing.T) {
	for _, item := range []struct {
		codes []uint8
		want  uint8
		note  string
	}{
		{[]uint8{0, 2, 5, 6}, 0, "牧師戰士法師賊：本作建得出來的全部"},
		{[]uint8{2, 4}, 1, "混進一個職業碼 4"},
		{[]uint8{0x0a}, 1, "職業碼 0Ah"},
		{nil, 0, "空隊伍"},
	} {
		if got := gamepack.PartySurpriseAlert(item.codes); got != item.want {
			t.Fatalf("%s: 得到 %d，預期 %d", item.note, got, item.want)
		}
	}
}

// `23h` 的兩個門檻配對是原版的樣子，而結果碼 3 到不了——寫下 3 之後
// 同一個條件會馬上把它改成 2。照抄那個行為，不「修正」成看起來合理的版本。
func TestSurpriseOutcomeFollowsTheOriginalBranches(t *testing.T) {
	// 門檻：隊伍 = 運算元4 + 2 − 運算元1，對方 = 運算元2 + 2 − 運算元3。
	operands := [gamepack.SurpriseOperands]uint8{1, 4, 1, 3} // 隊伍門檻 4，對方門檻 5
	for _, item := range []struct {
		partyRoll, foeRoll int
		want               uint8
		note               string
	}{
		{partyRoll: 6, foeRoll: 6, want: gamepack.SurpriseNeither, note: "兩邊都擲過門檻"},
		{partyRoll: 3, foeRoll: 6, want: gamepack.SurpriseParty, note: "隊伍沒過"},
		{partyRoll: 6, foeRoll: 4, want: gamepack.SurpriseFoes, note: "對方沒過"},
		{partyRoll: 3, foeRoll: 4, want: gamepack.SurpriseFoes, note: "兩邊都沒過，原版當成對方被突襲"},
	} {
		if got := gamepack.SurpriseOutcome(operands, item.partyRoll, item.foeRoll); got != item.want {
			t.Fatalf("%s: 得到 %d，預期 %d", item.note, got, item.want)
		}
	}
}
