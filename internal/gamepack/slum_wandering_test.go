package gamepack_test

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 走路遭遇（spec 136）。
//
// 原版每走一步跑一次命令集入口 1；貧民窟那一份先 `AND 7Fh` 取本格事件碼，
// 事件碼 0（普通格子）落到 `9AA5h`，那裡三道否決之後 `RANDOM 13`、`> 12`
// 才遭遇。所以**單走一步多半什麼都不會發生**——這條測試走很多步，釘的是
// 「走夠多步就一定遇得到」，以及遇到的東西不是空殼。
//
// 失敗的形狀有兩種：一路走到底一次都沒遇到（擲骰那段沒跑，或本格值沒投影
// 進去），或是主群的怪物編號不在 `B6E6h` 那張表的三個值裡（五張表沒查到）。
//
// **不要拿「編號 0」當失敗判準**——`B6E6h` 是 `00 02 04`，KOBOLDS 的編號
// 本來就是 0。擲到 KOBOLDS 才會踩到，所以那種斷言平常都綠的。
func TestSlumWalkingRollsAnEncounter(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	archive, err := gamepack.ReadDOSECLArchive(zipPath, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]gamepack.InitialCharacter, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, gamepack.InitialCharacter{
			Name: string(rune('A' + index)), ClassID: "fighter", CurrentHP: 60,
			Abilities: [6]int{18, 10, 10, 16, 10, 10},
		})
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9900, party...)
	if err != nil {
		t.Fatal(err)
	}
	// 一步一步走。步數挑 200：不加成時每步 1/14，200 步一次都不中的機率
	// 小於 10^-6，真的一次都沒中就是擲骰那段沒跑到。
	const steps = 200
	for step := 0; step < steps; step++ {
		position := gamepack.Spawn{X: uint8(14 - step%6), Y: 4}
		result, err := gamepack.RunInitialSessionSearchEntry(session, geometry.Grid{}, position)
		if err != nil {
			t.Fatalf("第 %d 步跑入口 1：%v", step, err)
		}
		for round := 0; round < 24 && len(result.MonsterSpawns) == 0 && !result.Exited; round++ {
			next, err := session.RunUntilEvent(4096, nil, true)
			if err != nil {
				t.Fatalf("第 %d 步第 %d 輪：%v", step, round, err)
			}
			result = next
		}
		if len(result.MonsterSpawns) == 0 {
			continue
		}
		t.Logf("第 %d 步擲中遭遇", step)
		main := result.MonsterSpawns[0]
		for index, spawn := range result.MonsterSpawns {
			t.Logf("  第 %d 群：怪物 %d、數量 %d、造形 %d",
				index, spawn.MonsterID, spawn.Count, spawn.IconBlock)
			if spawn.Count == 0 {
				t.Fatalf("第 %d 群的數量是 0——隊伍強度那一段沒算出來", index)
			}
			// 主群是數量最多的那一群：`LOAD MONSTER @6E80 @9808 @6E80`
			// 的數量直接是隊伍強度算出來的 `@9808`，另外兩群是 3、4 或 1。
			if spawn.Count > main.Count {
				main = spawn
			}
		}
		// `B6E6h` 的三格（KOBOLDS／GOBLINS／ORCS）。主群的編號只能是這三個
		// 之一；落在外面就表示 `GETTABLE` 讀錯位址，而那不會崩潰，只會安靜地
		// 排出一場錯的架。
		switch main.MonsterID {
		case 0, 2, 4:
		default:
			t.Fatalf("主群的怪物編號是 %d，不在 B6E6h 的 00/02/04 裡（spec 136）",
				main.MonsterID)
		}
		if main.IconBlock != main.MonsterID {
			t.Fatalf("主群造形 %d 與編號 %d 不同——`LOAD MONSTER @6E80 @9808 @6E80` 兩個位置都該是 @6E80",
				main.IconBlock, main.MonsterID)
		}
		return
	}
	t.Fatalf("走了 %d 步一次都沒遇到怪——`9B3Ah` 那一擲沒跑到（spec 136）", steps)
}

// 邊走邊搜會把怪引出來（spec 136）。
//
// `9B40h` 比對 `@6DCA == 1`，相等就把 `RANDOM 13` 那一擲加四——不搜是
// 1/14，搜是 5/14。`@6DCA` 就是命令列那兩個搜尋旗標（`[4937h]+594h`，
// 引擎在鍵盤分派裡 `xor ax,1` 切第 0 位、`or ax,2` 設第 1 位）。
//
// 這條釘的是**遊戲那一側要把旗標投影進 ECL**。沒投影的症狀不是崩潰，是
// 「按了 S 什麼都沒變」——玩起來只是覺得遭遇有點少。
//
// 釘的是「有差別」，不是「差多少」。實測 300 步是 12 比 45（4% 比 15%），
// 低於腳本的 1/14 比 5/14——遭遇之後 session 停在選單、狀態帶到下一步，
// 這裡沒有把戰鬥跑完。機率要對到小數點是對拍那一關的事。
func TestSearchingWhileWalkingRaisesTheEncounterRate(t *testing.T) {
	quiet := countSlumEncounters(t, 0)
	searching := countSlumEncounters(t, 1)
	t.Logf("不搜 %d 次、邊走邊搜 %d 次（各 300 步）", quiet, searching)
	if quiet == 0 {
		t.Fatal("不搜的那一輪一次都沒遇到——擲骰那段沒跑（spec 136）")
	}
	if searching <= quiet {
		t.Fatalf("邊走邊搜遇到 %d 次，沒有比不搜的 %d 次多——`@6DCA` 沒有被讀到",
			searching, quiet)
	}
}

// countSlumEncounters 走 300 步，回報擲中幾次。`searchFlags` 直接寫進
// `@6DCA`，也就是遊戲那一側該做的投影。
func countSlumEncounters(t *testing.T, searchFlags uint16) int {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	archive, err := gamepack.ReadDOSECLArchive(zipPath, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]gamepack.InitialCharacter, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, gamepack.InitialCharacter{
			Name: string(rune('A' + index)), ClassID: "fighter", CurrentHP: 60,
			Abilities: [6]int{18, 10, 10, 16, 10, 10},
		})
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9900, party...)
	if err != nil {
		t.Fatal(err)
	}
	encounters := 0
	for step := 0; step < 300; step++ {
		session.Machine().Memory[0x6DCA] = searchFlags
		position := gamepack.Spawn{X: uint8(14 - step%6), Y: 4}
		result, err := gamepack.RunInitialSessionSearchEntry(session, geometry.Grid{}, position)
		if err != nil {
			t.Fatalf("第 %d 步跑入口 1：%v", step, err)
		}
		for round := 0; round < 24 && len(result.MonsterSpawns) == 0 && !result.Exited; round++ {
			next, err := session.RunUntilEvent(4096, nil, true)
			if err != nil {
				t.Fatalf("第 %d 步第 %d 輪：%v", step, round, err)
			}
			result = next
		}
		if len(result.MonsterSpawns) > 0 {
			encounters++
		}
	}
	return encounters
}
