package treasure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 換算表照 START.EXE `DS:0D38h`（銅 1、銀 10、琥珀金 100、金 200、白金 1000），
// entry 11 的四捨五入是 `(總銅 + 100) ÷ 200`。
func TestGoldEquivalentRoundsLikeEntry11(t *testing.T) {
	var money [CurrencyCount]uint16
	money[Platinum], money[Gold], money[Silver], money[Copper] = 2, 3, 19, 199
	// 2000 + 600 + 190 + 199 = 2989 銅 → (2989 + 100) ÷ 200 = 15
	if got := GoldEquivalent(money); got != 15 {
		t.Fatalf("gold equivalent %d, want 15", got)
	}
	money[Copper] = 200 // 2990 → 3090 ÷ 200 = 15
	if got := GoldEquivalent(money); got != 15 {
		t.Fatalf("gold equivalent %d, want 15", got)
	}
	// 3010 銅 → (3010 + 100) ÷ 200 = 15；寶石與珠寶不算錢。
	money[Copper] = 210
	money[Gems], money[Jewelry] = 9, 9
	if got := GoldEquivalent(money); got != 15 {
		t.Fatalf("gold equivalent %d, want 15 (gems and jewelry do not count)", got)
	}
}

// entry 15：清五種硬幣，餘額重鑄成白金＋金。
func TestRemintWritesPlatinumAndGoldOnly(t *testing.T) {
	var money [CurrencyCount]uint16
	money[Copper], money[Silver], money[Gold], money[Gems] = 7, 7, 7, 3
	if err := RemintCharacterMoney(&money, 1234); err != nil {
		t.Fatal(err)
	}
	want := [CurrencyCount]uint16{}
	want[Platinum], want[Gold], want[Gems] = 246, 4, 3
	if money != want {
		t.Fatalf("money %v, want %v", money, want)
	}
}

// overlay-16 42C1h：從銅開始付、每種多付一枚、白金往下找零。
func TestPayInCoinsMatchesTheTrainingRoutine(t *testing.T) {
	// 只有 1000 金：金 1001 枚上限 1000，剩餘 0，不找零。
	var money [CurrencyCount]uint16
	money[Gold] = 1000
	if err := PayInCoins(&money, 1000); err != nil {
		t.Fatal(err)
	}
	if money != ([CurrencyCount]uint16{}) {
		t.Fatalf("money %v, want empty", money)
	}
	// 1500 金：付 1001 枚，多付 200 銅 → 找回 1 金。
	money = [CurrencyCount]uint16{}
	money[Gold] = 1500
	if err := PayInCoins(&money, 1000); err != nil {
		t.Fatal(err)
	}
	if money[Gold] != 500 || money[Platinum] != 0 {
		t.Fatalf("money %v, want 500 gold", money)
	}
	// 只有白金 500：付 201 枚，多付 1000 銅 → 找回 1 白金，淨扣 200。
	money = [CurrencyCount]uint16{}
	money[Platinum] = 500
	if err := PayInCoins(&money, 1000); err != nil {
		t.Fatal(err)
	}
	if money[Platinum] != 300 {
		t.Fatalf("money %v, want 300 platinum", money)
	}
	// 雜幣：銅 50、銀 30、金 3、白金 2 付 4 金（800 銅）：銅 50 全付（剩 750）、銀
	// 750÷10+1 = 76 → 30 枚（剩 450）、琥珀金 0、金 450÷200+1 = 3 → 3 枚（剩 −150）
	// → 找零從白金起：150÷1000 = 0、÷200 = 0、÷100 = 1 琥珀金（剩 50）、÷10 = 5 銀。
	money = [CurrencyCount]uint16{}
	money[Copper], money[Silver], money[Gold], money[Platinum] = 50, 30, 3, 2
	if err := PayInCoins(&money, 4); err != nil {
		t.Fatal(err)
	}
	want := [CurrencyCount]uint16{}
	want[Electrum], want[Silver], want[Platinum] = 1, 5, 2
	if money != want {
		t.Fatalf("money %v, want %v", money, want)
	}
	// 四捨五入的縫：999 金 199 銅過得了 1000 金的門（entry 11 算 1000），五種硬幣付完
	// 還差 1 銅——原版往表外讀，remake 收光不找零。
	money = [CurrencyCount]uint16{}
	money[Gold], money[Copper] = 999, 199
	if GoldEquivalent(money) != 1000 {
		t.Fatalf("999 gold 199 copper should round to 1000, got %d", GoldEquivalent(money))
	}
	if err := PayInCoins(&money, 1000); err != nil {
		t.Fatal(err)
	}
	if money != ([CurrencyCount]uint16{}) {
		t.Fatalf("money %v, want everything taken", money)
	}
}

// 神殿／武具店：角色夠就角色出（餘額重鑄），不夠才 pool，不混付。
func TestPayGoldPrefersTheCharacterThenThePool(t *testing.T) {
	state := poolsave.State{Party: []poolsave.Character{{Name: "A"}}, CharacterLibrary: []poolsave.Character{{Name: "A"}}}
	state.Party[0].Money[Platinum] = 100 // 500 金
	state.PooledMoney[Gold] = 1000
	source, ok, err := PayGold(&state, 0, 120)
	if err != nil || !ok || source != PaidByCharacter {
		t.Fatalf("source %q ok %v err %v", source, ok, err)
	}
	if state.Party[0].Money[Platinum] != 76 || state.Party[0].Money[Gold] != 0 || state.PooledMoney[Gold] != 1000 {
		t.Fatalf("after paying 120: %v pool %v", state.Party[0].Money, state.PooledMoney)
	}
	if state.CharacterLibrary[0].Money != state.Party[0].Money {
		t.Fatal("library copy was not synced")
	}
	source, ok, err = PayGold(&state, 0, 500)
	if err != nil || !ok || source != PaidByPool {
		t.Fatalf("source %q ok %v err %v", source, ok, err)
	}
	if state.PooledMoney[Platinum] != 100 || state.PooledMoney[Gold] != 0 || state.Party[0].Money[Platinum] != 76 {
		t.Fatalf("after the pool paid 500: %v pool %v", state.Party[0].Money, state.PooledMoney)
	}
	if _, ok, err := PayGold(&state, 0, 9999); err != nil || ok {
		t.Fatalf("ok %v err %v paying more than both have", ok, err)
	}
}

// dosgolem 收據（docs/audit/dosgolem-training-fee.json，#29）：同一個角色注入三種錢包
// 走到盜賊門前按 t、y，原版記錄的五個錢欄前後——remake 的 PayInCoins 要給同一個答案，
// 錢不夠那一筆要被 GoldEquivalent 擋下。
func TestPayInCoinsMatchesTheDosgolemTrainingReceipt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-training-fee.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Scenarios map[string]struct {
			Frames []struct {
				Label  string         `json:"label"`
				Wallet map[string]int `json:"wallet"`
				Level  int            `json:"thief_level"`
			} `json:"frames"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	names := []string{"copper", "silver", "electrum", "gold", "platinum"}
	toMoney := func(wallet map[string]int) [CurrencyCount]uint16 {
		var money [CurrencyCount]uint16
		for coin, name := range names {
			money[coin] = uint16(wallet[name])
		}
		return money
	}
	for name, scenario := range receipt.Scenarios {
		before, after := scenario.Frames[0], scenario.Frames[len(scenario.Frames)-1]
		money := toMoney(before.Wallet)
		affordable := GoldEquivalent(money) >= 1000
		if affordable != (after.Level == 2) {
			t.Fatalf("%s: remake says affordable=%v but the original %s", name, affordable, map[bool]string{true: "trained", false: "refused"}[after.Level == 2])
		}
		if !affordable {
			if toMoney(after.Wallet) != money {
				t.Fatalf("%s: the original changed the wallet on a refusal: %v", name, after.Wallet)
			}
			continue
		}
		if err := PayInCoins(&money, 1000); err != nil {
			t.Fatal(err)
		}
		if want := toMoney(after.Wallet); money != want {
			t.Fatalf("%s: remake wallet after paying 1000 gold %v, original %v", name, money, want)
		}
		t.Logf("%s: %v → %v matches the original", name, before.Wallet, after.Wallet)
	}
	if len(receipt.Scenarios) != 3 {
		t.Fatalf("receipt has %d scenarios, want 3", len(receipt.Scenarios))
	}
}
