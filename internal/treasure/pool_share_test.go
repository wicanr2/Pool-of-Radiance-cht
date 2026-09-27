package treasure

// 公款與錢的邊角（spec 067〈公款〉、spec 040〈Share〉，issue #79）。

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 付款的餘額只留低位字：entry 15／16 是 `retf 2`，推進去的是 32 位元餘額的低位字。
func TestPayGoldTruncatesTheRemainderToSixteenBits(t *testing.T) {
	state := poolsave.State{Party: []poolsave.Character{{Name: "A"}}}
	state.PooledMoney[Platinum] = 20000 // 100000 金
	source, ok, err := PayGold(&state, 0, 10)
	if err != nil || !ok || source != PaidByPool {
		t.Fatalf("source %q ok %v err %v", source, ok, err)
	}
	// 99990 & FFFFh = 34454 → 白金 6890、金 4。
	if state.PooledMoney != [CurrencyCount]uint32{Gold: 4, Platinum: 6890} {
		t.Fatalf("pool after paying 10 out of 100000: %v", state.PooledMoney)
	}
}

// 商店與神殿只收 entry 11 的低位字（`03B4h`、`0177h`）：13200 白金＝66000 金，低位字 464，
// 付不起 500，改由公款出；鑑定比的是完整的 32 位元（`1FDDh`／`1FE0h`），角色自己付，
// 餘額 65500 剛好放得進一個字。
func TestShopComparesTheLowWordButIdentifyComparesTheWholeValue(t *testing.T) {
	rich := func() poolsave.State {
		state := poolsave.State{Party: []poolsave.Character{{Name: "A"}}}
		state.Party[0].Money[Platinum] = 13200
		state.PooledMoney[Gold] = 1000
		return state
	}
	state := rich()
	source, ok, err := PayGold(&state, 0, 500)
	if err != nil || !ok || source != PaidByPool || state.Party[0].Money[Platinum] != 13200 {
		t.Fatalf("shop: source %q ok %v err %v money %v", source, ok, err, state.Party[0].Money)
	}
	state = rich()
	source, ok, err = PayGoldFullCharacter(&state, 0, 500)
	if err != nil || !ok || source != PaidByCharacter {
		t.Fatalf("identify: source %q ok %v err %v", source, ok, err)
	}
	if state.Party[0].Money != [CurrencyCount]uint16{Platinum: 13100} || state.PooledMoney[Gold] != 1000 {
		t.Fatalf("identify: money %v pool %v", state.Party[0].Money, state.PooledMoney)
	}
}

func npcMember(name string, control uint8) poolsave.Character {
	member := moneyCharacter(name, 18)
	member.NPC = true
	member.Record = make([]byte, poolsave.NPCRecordSize)
	member.Record[0x84] = control
	return member
}

// NPC（`+84h` 80h 以上、不是 B3h）不進公款、不算份數、第一輪不發（`052Bh`、`05F9h`、`0721h`）。
func TestPoolAndShareLeaveTheNPCOut(t *testing.T) {
	hero, npc := moneyCharacter("A", 18), npcMember("NPC", 0x85)
	hero.Money[Gold], npc.Money[Gold] = 10, 7
	state := moneyState(hero, npc)
	if err := PoolMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.PooledMoney[Gold] != 10 || state.Party[1].Money[Gold] != 7 {
		t.Fatalf("pool %v, NPC wallet %v", state.PooledMoney, state.Party[1].Money)
	}
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.Party[0].Money[Gold] != 10 || state.Party[1].Money[Gold] != 7 || state.PooledMoney[Gold] != 0 {
		t.Fatalf("hero %v NPC %v pool %v", state.Party[0].Money, state.Party[1].Money, state.PooledMoney)
	}
}

// B3h（死靈術叫起來的）算進份數、第一輪卻不發：他那一份從公款消失。
func TestShareDropsTheAnimatedMembersShare(t *testing.T) {
	state := moneyState(moneyCharacter("A", 18), npcMember("Z", 0xb3))
	state.PooledMoney[Gold] = 10
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.Party[0].Money[Gold] != 5 || state.Party[1].Money[Gold] != 0 || state.PooledMoney[Gold] != 0 {
		t.Fatalf("A %v Z %v pool %v", state.Party[0].Money, state.Party[1].Money, state.PooledMoney)
	}
}

// 第二輪（`087Ah`）把第一輪發不出去的沿整隊再發給有空間的人；餘數那一枚要先過
// 「現重＋整筆餘數」的容量檢查（`07B6h`）。
func TestShareGivesTheLeftoverToWhoeverHasRoom(t *testing.T) {
	full := moneyCharacter("A", 3) // 上限 1150
	full.Money[Copper] = 1150
	state := moneyState(full, moneyCharacter("B", 18))
	state.PooledMoney[Gold] = 3
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.Party[0].Money[Gold] != 0 || state.Party[1].Money[Gold] != 3 || state.PooledMoney[Gold] != 0 {
		t.Fatalf("A %v B %v pool %v", state.Party[0].Money, state.Party[1].Money, state.PooledMoney)
	}
}

// 超重的人：「上限 − 現重」以 16 位元繞回。原版收據（spec 067〈S 沒分下去〉）裡現重 65486、
// 上限 1700 的人按 S 拿到 1750 顆珠寶、白金一枚也沒拿；這裡用一件比上限重 50 的物品重現同一個形狀。
func TestShareWrapsAroundForAnOverloadedMember(t *testing.T) {
	member := moneyCharacter("A", 3) // 上限 1150
	heavy := make([]byte, 63)
	binary.LittleEndian.PutUint16(heavy[0x37:0x39], 1200)
	member.Inventory = []poolsave.Item{{Name: "ANVIL", Raw: heavy}}
	state := moneyState(member)
	state.PooledMoney[Platinum] = 100
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.Party[0].Money[Jewelry] != 65486 || state.Party[0].Money[Platinum] != 0 {
		t.Fatalf("wallet %v", state.Party[0].Money)
	}
	if state.PooledMoney[Jewelry] != 50 || state.PooledMoney[Platinum] != 100 {
		t.Fatalf("pool %v", state.PooledMoney)
	}
}

// dosgolem 收據 docs/audit/dosgolem-shop-share-wrap.json：P 之後現重 65486、上限 1700（力量 14）
// 的人按 S。現重用一件重 65486 的物品重現，錢包與公款七欄逐欄比對 `s` 那一幀。
func TestShareMatchesTheDosgolemWrapReceipt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-shop-share-wrap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Frames []struct {
			Label  string            `json:"label"`
			Wallet map[string]uint32 `json:"wallet"`
			Pool   map[string]uint32 `json:"pool"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	coins := []string{"copper", "silver", "electrum", "gold", "platinum", "gems", "jewelry"}
	if len(receipt.Frames) < 3 || receipt.Frames[1].Label != "p" || receipt.Frames[2].Label != "s" {
		t.Fatalf("receipt frames %+v", receipt.Frames)
	}
	before, after := receipt.Frames[1], receipt.Frames[2]
	member := moneyCharacter("HUMAN", 14)
	heavy := make([]byte, 63)
	binary.LittleEndian.PutUint16(heavy[0x37:0x39], uint16(before.Wallet["load"]))
	member.Inventory = []poolsave.Item{{Name: "LOAD", Raw: heavy}}
	state := moneyState(member)
	for index, coin := range coins {
		state.Party[0].Money[index] = uint16(before.Wallet[coin])
		state.PooledMoney[index] = before.Pool[coin]
	}
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	for index, coin := range coins {
		if uint32(state.Party[0].Money[index]) != after.Wallet[coin] || state.PooledMoney[index] != after.Pool[coin] {
			t.Errorf("%s: wallet %d pool %d, receipt %d / %d", coin, state.Party[0].Money[index],
				state.PooledMoney[index], after.Wallet[coin], after.Pool[coin])
		}
	}
}
