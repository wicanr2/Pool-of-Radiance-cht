package main

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 底列照公款有沒有錢選 overlay-06 的兩個字串（`0487h`／`0460h`，spec 164）。
func TestShopMenuCommandsFollowThePool(t *testing.T) {
	a := newShopApp(t)
	want := []messageID{msgShopCommandBuy, msgShopCommandView, msgShopCommandPool,
		msgShopCommandAppraise, msgShopCommandExit}
	if got := a.shopMenuCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("empty pool: %v, want Buy View Pool Appraise Exit", got)
	}
	a.state.PooledMoney[pooltreasure.Gold] = 5
	want = []messageID{msgShopCommandBuy, msgShopCommandView, msgShopCommandTake,
		msgShopCommandPool, msgShopCommandShare, msgShopCommandAppraise, msgShopCommandExit}
	if got := a.shopMenuCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pool with money: %v, want Buy View Take Pool Share Appraise Exit", got)
	}
	a.language = languageEnglish
	if got := a.text(msgShopCommandAppraise); got != "Appraise" {
		t.Fatalf("English label %q", got)
	}
}

// 主選單上 ENTER 與方向鍵不買東西；B 開清單，清單裡 ESC 回主選單，
// 主選單的 E 與 ESC 離店。全程從 Update() 送鍵。
func TestShopMenuKeysOpenAndCloseTheBuyList(t *testing.T) {
	a := newShopApp(t)
	pressKeys(t, a, ebiten.KeyDown, ebiten.KeyEnter)
	if len(a.state.Party[0].Inventory) != 0 || a.shop.buying {
		t.Fatalf("ENTER on the shop menu bought %v (buying %v)", a.state.Party[0].Inventory, a.shop.buying)
	}
	pressKeys(t, a, ebiten.KeyB)
	if !a.shop.buying {
		t.Fatal("B did not open the list of wares")
	}
	pressKeys(t, a, ebiten.KeyEnter)
	if len(a.state.Party[0].Inventory) != 1 {
		t.Fatalf("ENTER in the list did not buy (%q)", a.shop.message)
	}
	pressKeys(t, a, ebiten.KeyEscape)
	if a.shop == nil || a.shop.buying || !a.shopActive {
		t.Fatal("ESC in the list should return to the shop menu")
	}
	pressKeys(t, a, ebiten.KeyE)
	if a.shopActive {
		t.Fatal("E on the shop menu did not leave the shop")
	}
}

// 店主肖像取最近一次 PICTURE：head 是當時的 `6DE1h`、body 是運算元。
// `PICS` 關掉、PIC 那一條（`6DE1h == FFh`）或載不到都退回視野。
func TestShopkeeperPortraitComesFromTheLastPicture(t *testing.T) {
	a := newShopApp(t)
	a.spawn.Map.Archive = 3
	var asked [3]uint8
	calls := 0
	portrait := ebiten.NewImage(88, 88)
	a.loadNPCPortrait = func(archive, head, body uint8) (*ebiten.Image, error) {
		asked, calls = [3]uint8{archive, head, body}, calls+1
		return portrait, nil
	}
	a.lastPicture = eclPicture{head: 42, value: 9, set: true}
	if got := a.shopkeeperPortrait(); got != portrait || asked != [3]uint8{3, 42, 9} {
		t.Fatalf("portrait %v asked %v, want HEAD3/42 + BODY3/9", got, asked)
	}
	if a.shopkeeperPortrait(); calls != 1 {
		t.Fatalf("the portrait was loaded %d times; it should be cached", calls)
	}
	a.clearNPCPortrait()
	a.portraitsHidden = true
	if a.shopkeeperPortrait() != nil {
		t.Fatal("Portraits off still drew the shopkeeper")
	}
	a.portraitsHidden = false
	a.shop.buying = true
	if a.shopkeeperPortrait() != nil {
		t.Fatal("the list of wares still drew the shopkeeper")
	}
	a.shop.buying = false
	a.clearNPCPortrait()
	a.lastPicture = eclPicture{head: 0xFF, value: 9, set: true}
	if a.shopkeeperPortrait() != nil || calls != 1 {
		t.Fatal("the PIC path (6DE1h = FFh) should fall back to the view")
	}
}
