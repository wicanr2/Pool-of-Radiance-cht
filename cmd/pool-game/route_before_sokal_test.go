package main

// 索寇要塞之前的路線與休息（goal `docs/goals/issue-22-26-levels-before-sokal.md`）。
//
// 古托井西緣是陸路去波多廣場的那一步（#26 建議順序第 3 段的前提）。
//
// `ecl8/29 9947h` 以朝向 `@C04D` 查換圖表 `@AFCA`（區塊）與 `@AFC7`（封存檔）：
// `01 14 0F 12`／`10 02 02 01` → 北是不存在的封存檔 10h（當牆，#22）、東 ecl2/20、
// 南 ecl2/15、西 ecl1/18（spec 101，exact）。反方向是 `ecl1/18 99D4h ON GOTO @C04D`
// 第 1 支 `99F0h`：`NEWECL 29`。所以波多廣場在索寇要塞之前就走得到——東航線
// （`4AA7 >= FEh` 才開）不是唯一的路。
//
// 從標題正常開局，全程從 `Update()` 送鍵；隊伍是 `bootCityParty` 的六名 60 HP 矮人
// 穿上鏈甲、盾與長劍，這條測的是路通不通，不是一級隊伍打不打得過。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestKutoWellWestEdgeLeadsToPodolPlaza(t *testing.T) {
	walkToPodolPlaza(t, "")
}

// walkToPodolPlaza 從標題開局走到波多廣場，回傳站在 ECL1/18 的駕駛。entry 是進場
// 那個選單（`9A36h`）要答的項目，空字串就照 `settle` 的預設。換圖分兩拍：ECL 先換成
// 1/18，選單答完才載入 GEO1/18。
func walkToPodolPlaza(t *testing.T, entry string) *mainlineDriver {
	t.Helper()
	application := bootCityParty(t, dosZIPForTests)
	// 沒穿甲的隊伍在貧民窟會被放大過的遭遇打光（`equipForFirstCombat` 的註解）。
	application.state.Party = equipForFirstCombat(t, application.state.Party)
	d := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{}, tolerateDefeat: false,
		step: func(key ebiten.Key) {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}}
	a := d.a
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		t.Skipf("開場沒有停在城區：ECL%d/%d", a.eclArchive, a.eventSession.CurrentBlockID())
	}
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	d.leaveMap("city → west", func(x, y int) bool { return x == 0 && y == 4 }, street, false, 3)
	if a.spawn.Map != (gamepack.MapKey{Archive: 2, BlockID: 20}) {
		t.Fatalf("城區 (0,4) 往西到了 %+v，該是貧民窟", a.spawn.Map)
	}
	d.crossSlums(false)
	if a.spawn.Map != (gamepack.MapKey{Archive: 8, BlockID: 29}) {
		t.Fatalf("貧民窟西緣到了 %+v，該是古托井", a.spawn.Map)
	}
	// 最後一步自己踏、不經過 `settle`：波多廣場一進來就問怎麼進場（`9A36h`），
	// `settle` 會替玩家答第 0 項 STRIDE BOLDLY FORWARD!，那一項看不到拍賣。
	westEdge := func(x, y int) bool { return x == 0 && d.a.initialMap.Grid.CanMoveDungeonWrapped(x, y, 6) }
	if !westEdge(int(a.spawn.X), int(a.spawn.Y)) && !d.walkAllowing("Kuto's Well west edge", westEdge, d.kutoSafe, false) {
		t.Fatalf("走不到古托井西緣，停在 %+v", a.spawn)
	}
	d.settle()
	d.face(3)
	d.step(ebiten.KeyArrowUp)
	for guard := 0; guard < 16 && a.cellEventPending && !a.cellWaitingMenu; guard++ {
		d.step(ebiten.KeyEnter)
	}
	if entry != "" && a.cellWaitingMenu && len(a.cellMenuOptions) > 1 {
		if err := selectMenuOption(t, a, entry); err != nil {
			t.Fatalf("進場選 %q：%v（%v %q）", entry, err, a.cellMenuOptions, a.eventText)
		}
	}
	d.settle()
	if a.spawn.Map != (gamepack.MapKey{Archive: 1, BlockID: 18}) || a.eclArchive != 1 ||
		a.eventSession.CurrentBlockID() != 18 {
		t.Fatalf("古托井西緣到了 %+v（ECL%d/%d），`9947h` 的表西向是 ecl1/18",
			a.spawn.Map, a.eclArchive, a.eventSession.CurrentBlockID())
	}
	if a.eventMachine.Memory[0x4AA7] >= 0xFE {
		t.Fatalf("4AA7=%02X：這條要證明的是索寇之前就走得到", a.eventMachine.Memory[0x4AA7])
	}
	t.Logf("Podol Plaza at (%d,%d) facing %d", a.spawn.X, a.spawn.Y, a.spawn.Facing)
	return d
}

// 波多廣場的拍賣要先在市政廳接到委任才會開（`ecl1/18`，exact 靜態）：
//
//	入口 4 `9971h`（換圖載入時跑，比進場選單晚一拍）
//	998D  COMPARE @4AB0, 1 ; IF <> ; GOTO 99A5
//	9998  SAVE 64 → @4A34 ; SAVE 0 → @4A35 ; EXIT      ; 接了委任：拍賣開
//	99A5  COMPARE @4AE7, 254 ; IF < ; SAVE 0 → @4A34
//	99B2  SAVE 255 → @4A35 ; EXIT                       ; 沒接：拍賣不開
//
// 槽 10 由市政廳職員 `ecl3/8 AA2Ah` 列出委任時寫 1。職員的清單（`A83Fh` 起）從索引 0
// 開始、跳過已完成（槽 255）的、一次最多列三條，順序是貧民窟、索寇要塞、書、波多廣場
// ——**前三條至少完成一條，波多廣場才會被列出來**。
//
// 接到之後，拍賣不打大場也結得了案：進場選 DISGUISE PARTY AS MONSTERS.（`9A36h` →
// `GETTABLE @B255` 得 1）→ 地形碼 1 → STAND AND LISTEN → WAIT FOR WINNER →
// `A60Ch` "GOING... GONE!" → `A865h SAVE 254 @4A35`、`A86Bh SAVE 254 @4AB0`。從 `A60Ch`
// 沿控制流走到底（含 GOSUB 與 IF 的跳過）12 條指令、沒有 `COMBAT`；對照組 LEAVE
// （`A67Ah`）可達 `9DEDh` 的一場。那一半沒有實跑收據——接委任要先交貧民窟的件。
//
// 這一條釘住閘門：沒接委任時，進場選喬裝也拿不到拍賣（選單寫的 1 被入口 4 蓋成 FF）。
func TestPodolAuctionNeedsTheCommissionFirst(t *testing.T) {
	d := walkToPodolPlaza(t, "DISGUISE PARTY AS MONSTERS.")
	a := d.a
	if got := a.eventMachine.Memory[0x4AB0]; got == 1 {
		t.Fatalf("4AB0=1：這條要的是沒接委任的狀態")
	}
	if got := a.eventMachine.Memory[0x4A35]; got != 255 {
		t.Fatalf("沒接委任時 4A35=%02X，入口 4 `99B2h` 該寫 FF（拍賣不開）", got)
	}
	// 負對照：選單本身有照選項寫——`6E79` 是喬裝那一項。
	if got := a.eventMachine.Memory[0x6E79]; got != 1 {
		t.Fatalf("進場選單的選擇 6E79=%d，喬裝是索引 1", got)
	}
}

// 貧民窟的屋內睡得成（`ecl2/20 9A24h`：`@6E82 != 0` → 0／0），而駕駛只挑走進去不會開打
// 的那幾格（`slumsIndoor`）。從標題開局走進貧民窟，負對照先在街上量 24／24。
func TestSlumsQuietRoomLetsThePartyRest(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	application.state.Party = equipForFirstCombat(t, application.state.Party)
	d := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}}
	a := d.a
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	d.leaveMap("city → west", func(x, y int) bool { return x == 0 && y == 4 }, street, false, 3)
	if a.spawn.Map != slumsMap {
		t.Fatalf("城區 (0,4) 往西到了 %+v，該是貧民窟", a.spawn.Map)
	}
	if code := d.terrainCode(int(a.spawn.X), int(a.spawn.Y)); code != 0 {
		t.Fatalf("進貧民窟那一格的地形碼是 %d，負對照要站在街上", code)
	}
	if err := a.runRestEntry(); err != nil {
		t.Fatal(err)
	}
	if got := a.restInterruption(); got.Period != 24 || got.Threshold != 24 {
		t.Fatalf("貧民窟街上的打斷設定是 %d／%d，`9A3Ch` 是 24／24", got.Period, got.Threshold)
	}
	// 還沒打過的戰鬥房不算：地形碼 1（獸人爭文件）的旗標 `4ACA` 還是 0。
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if d.terrainCode(x, y) == 1 && d.slumsIndoor(x, y) {
				t.Fatalf("(%d,%d) 地形碼 1 還沒打過（4ACA=%02X），不該是休息的候選",
					x, y, a.eventMachine.Memory[0x4ACA])
			}
		}
	}
	if !d.restIndoors() {
		t.Fatalf("走不進安靜的屋內，停在 (%d,%d) 地形碼 %d", a.spawn.X, a.spawn.Y,
			d.terrainCode(int(a.spawn.X), int(a.spawn.Y)))
	}
	code := d.terrainCode(int(a.spawn.X), int(a.spawn.Y))
	switch code {
	case 4, 7, 11, 17:
	default:
		t.Fatalf("休息的那一格地形碼是 %d，開局時只該挑本來就安靜的 4／7／11／17", code)
	}
	if got := a.restInterruption(); got.Period != 0 || got.Threshold != 0 {
		t.Fatalf("屋內 (%d,%d) 地形碼 %d 的打斷設定是 %d／%d，該是 0／0",
			a.spawn.X, a.spawn.Y, code, got.Period, got.Threshold)
	}
	if a.tactical != nil || a.gameOver {
		t.Fatalf("走進安靜的屋內卻開打了：%q", a.eventText)
	}
	t.Logf("rest indoors at (%d,%d) terrain %d", a.spawn.X, a.spawn.Y, code)
}
