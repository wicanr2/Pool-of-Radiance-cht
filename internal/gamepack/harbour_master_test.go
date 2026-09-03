package gamepack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// harbourMasterSession 建一個城區（ECL3 block 0）的 session，並把主線旗標
// 擺成指定的狀態。`4A01` 是船票、`4AA7` 是索寇要塞打完沒有（spec 102）。
func harbourMasterSession(t *testing.T, ticket, routes uint16) (*eclvm.BlockSession, GeometryMap) {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	archive, err := ReadDOSECLArchive(zipPath, 3)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := catalog.MapByBlock(0)
	if !ok {
		t.Fatal("GEO block 0 not found")
	}
	session, err := NewCellSweepSession(archive, 0, harbourParty()...)
	if err != nil {
		t.Fatal(err)
	}
	session.Machine().Memory[0x4A01] = ticket
	session.Machine().Memory[0x4AA7] = routes
	// 入口 1 一開頭就是 `9A00h COMPARE @4AC5, 1 ; IF < ; EXIT`：
	// `4AC5` 沒開起來的話，城區的地點分派整支不跑。
	session.Machine().Memory[0x4AC5] = 1
	return session, geoMap
}

func harbourParty() []InitialCharacter {
	party := make([]InitialCharacter, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, InitialCharacter{
			Name:          string(rune('A' + index)),
			ClassID:       "fighter",
			Abilities:     [6]int{18, 10, 10, 16, 10, 10},
			CurrentHP:     60,
			// 士氣要小於 128：`A1C1h COMPARE @6BB8, 128 ; IF <` 才會往付錢
			// 那一支走，128 以上是 NPC，港務長會說「THAT PERSON WON'T PAY.」。
			ControlMorale: 0,
		})
	}
	return party
}

// runHarbourMaster 走「往北踏進 (11,1)」這一步，回傳移動後入口 1 跑出來的結果。
func runHarbourMaster(t *testing.T, session *eclvm.BlockSession, geoMap GeometryMap,
	selections []uint16) eclvm.Result {
	t.Helper()
	spawn := Spawn{Map: geoMap.Key, X: 11, Y: 1, Facing: 0}
	if _, err := RunInitialSessionCellEntry(session, geoMap.Grid, spawn); err != nil {
		t.Fatal(err)
	}
	if err := session.SetEntry(1); err != nil {
		t.Fatal(err)
	}
	// 一次 RunUntilEvent 只跑到下一個事件（印字也算），所以要一路跑到
	// 停在選單上或腳本結束為止。
	last := eclvm.Result{}
	for round := 0; round < 64; round++ {
		result, err := session.RunUntilEvent(8192, selections, true)
		if err != nil {
			t.Fatal(err)
		}
		last.Events = append(last.Events, result.Events...)
		last.Menus = append(last.Menus, result.Menus...)
		last.WaitingForMenu = result.WaitingForMenu
		last.Exited = result.Exited
		if result.WaitingForMenu || result.Exited {
			return last
		}
		if len(result.Events) == 0 && len(result.Menus) == 0 {
			return last
		}
	}
	return last
}

// 港務長要面向北才會開口，而且手上有票（`4A01 == 1`）時不再說話（spec 102）。
func TestHarbourMasterStaysSilentWhileTheTicketIsHeld(t *testing.T) {
	session, geoMap := harbourMasterSession(t, 1, 254)
	result := runHarbourMaster(t, session, geoMap, nil)
	for _, menu := range result.Menus {
		for _, option := range menu.Options {
			if option == "SOKAL" {
				t.Fatalf("手上還有票就不該出現航線選單：%v", menu.Options)
			}
		}
	}
}

// 要塞打完（`4AA7 = 254`）而且票被清成 255 之後，港務長給的是完整航線選單。
func TestHarbourMasterOffersEveryRouteOnceTheKeepIsCleared(t *testing.T) {
	session, geoMap := harbourMasterSession(t, 255, 254)
	result := runHarbourMaster(t, session, geoMap, nil)
	if !result.WaitingForMenu {
		t.Fatalf("沒有停在選單上：events=%d menus=%d", len(result.Events), len(result.Menus))
	}
	want := []string{"SOKAL", "EAST", "WEST", "BAY", "NONE"}
	menu := result.Menus[len(result.Menus)-1]
	if len(menu.Options) != len(want) {
		t.Fatalf("選單是 %v，要 %v", menu.Options, want)
	}
	for index, option := range want {
		if menu.Options[index] != option {
			t.Fatalf("選單第 %d 項是 %q，要 %q（%v）", index, menu.Options[index], option, menu.Options)
		}
	}
}

// 選單存的是 0 起算的游標，直接寫進 `4AC4`；碼頭再拿它分派目的地（spec 102）。
func TestHarbourMasterStoresTheChosenDestination(t *testing.T) {
	for index, name := range []string{"SOKAL", "EAST", "WEST", "BAY"} {
		session, geoMap := harbourMasterSession(t, 255, 254)
		runHarbourMaster(t, session, geoMap, []uint16{uint16(index)})
		if got := session.Machine().Memory[0x4AC4]; got != uint16(index) {
			t.Fatalf("選 %s 之後 4AC4 = %d，要 %d", name, got, index)
		}
	}
}

// 碼頭 (15,1) 拿 `4AC4` 分派：0 是索寇要塞（`NEWECL 21`），其餘三個是野外
// 圖 26／27 的固定座標（spec 102 的表）。
func TestPierSendsTheBoatWhereTheTicketSays(t *testing.T) {
	for _, row := range []struct {
		name        string
		destination uint16
		x, y, area  uint16
	}{
		{"SOKAL", 0, 0, 0, 4},
		{"EAST", 1, 9, 29, 8},
		{"WEST", 2, 7, 29, 7},
		{"BAY", 3, 13, 27, 7},
	} {
		session, geoMap := harbourMasterSession(t, 1, 254)
		machine := session.Machine()
		machine.Memory[0x4AC4] = row.destination
		spawn := Spawn{Map: geoMap.Key, X: 15, Y: 1, Facing: 0}
		if _, err := RunInitialSessionCellEntry(session, geoMap.Grid, spawn); err != nil {
			t.Fatal(err)
		}
		if err := session.SetEntry(1); err != nil {
			t.Fatal(err)
		}
		// `NEWECL` 會換到別的 archive 的區塊，這個 session 沒有跨檔的
		// resolver，所以換區那一步失敗是預期的；要看的是換區之前寫下的值。
		// 上船會先跳一個「按下 RETURN」的提示（`9BD2h GOSUB AF1Ch`），
		// 每一輪都給它一個選擇，免得停在那裡。
		for round := 0; round < 64; round++ {
			result, err := session.RunUntilEvent(8192, []uint16{0}, true)
			if err != nil {
				break
			}
			if result.Exited {
				break
			}
			if len(result.Events) == 0 && len(result.Menus) == 0 {
				break
			}
		}
		if got := machine.Memory[0x6E12]; got != row.area {
			t.Errorf("%s：6E12 = %d，要 %d", row.name, got, row.area)
		}
		if row.destination == 0 {
			// 索寇要塞是 `NEWECL 21`，區塊在 ECL4；這個 session 只掛了
			// ECL3，換不過去。`6E12 = 4` 是換區之前寫下的，足以認出這一支。
			continue
		}
		if got := machine.Memory[0x49C3]; got != row.x {
			t.Errorf("%s：49C3 = %d，要 %d", row.name, got, row.x)
		}
		if got := machine.Memory[0x49C4]; got != row.y {
			t.Errorf("%s：49C4 = %d，要 %d", row.name, got, row.y)
		}
	}
}

// 選 NONE 不付錢也不拿票。
func TestHarbourMasterLeavesTheTicketAloneOnNone(t *testing.T) {
	session, geoMap := harbourMasterSession(t, 255, 254)
	runHarbourMaster(t, session, geoMap, []uint16{4})
	if got := session.Machine().Memory[0x4A01]; got == 1 {
		t.Fatalf("選 NONE 之後 4A01 = %d，不該買到票", got)
	}
}
