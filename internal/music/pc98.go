package music

import "time"

// PC-98 版（Pony Canyon 1989）的派曲規則（spec 169）。
//
// PC-98 版是四個平台裡唯一全程有配樂的：15 首 YM2203 曲子，**每一個 ECL 區塊
// 都配一首**。下面每一條都讀自那一版的 `GAME.EXE`／`GAME.OVR`（SHA-256 見
// spec 169），位址是載入映像（MZ 標頭去除後）或 `GAME.OVR` 的檔案偏移。

// Source 是曲子的來源。
type Source string

const (
	// SourcePC98 是 PC-98 版的 15 首（本機 full-local 的預設）。
	SourcePC98 Source = "pc98"
	// SourceAmiga 是 Amiga 版的 6 首（spec 128）。
	SourceAmiga Source = "amiga"
)

// PC-98 版非區域的曲號（1 起算，與 `GAME.EXE $64E6` 的參數相同）。
const (
	// PC98SongTitle：overlay 1（`GAME.OVR 00D2Fh`：`mov al,1 ; call 062C:0226`），
	// 標題圖載入之後。
	PC98SongTitle = 1
	// PC98SongCombat／PC98SongBossCombat：overlay 10 戰場建構的最後一段
	//（`GAME.OVR 0E9D1h`）：先把模式設成 5，`[9D41h] == 42h` 放 12，否則放 11。
	PC98SongCombat     = 11
	PC98SongBossCombat = 12
	// PC98SongShop：overlay 6（商店，`GAME.OVR 078C0h`）模式設 1 之後放 14。
	PC98SongShop = 14
	// PC98SongEnding：overlay 18（結局，`GAME.OVR 22825h`）模式設 7 之後放 15。
	PC98SongEnding = 15
)

// PC98BossMonsterID 是讓戰鬥改放第 12 首的怪物記錄編號。
//
// `[9D41h]` 只有一個寫入點：overlay 17 的怪物記錄載入常式（`GAME.OVR 1FBE9h`
// `mov al,[bp+0Ah] ; mov [9D41h],al`），參數同時交給 `MONCHA.DAX` 的讀檔
// （`call 04C4:0DC9`），並複製 `11Dh`（285）位元組——就是一筆怪物記錄。
// 所以 `[9D41h]` 是**最後載入的那一筆怪物記錄編號**。
//
// `42h` ＝ 66：PC-98 `MONCHA.DAX` block 66 的名字是半形片假名
// `ﾃｨﾗﾝｽﾗｸｻｽ`（ティランスラクサス），DOS `MON5CHA.DAX` block 66 是
// `TYRANITHRAXUS`——最後一戰的提蘭斯拉克蘇斯。
const PC98BossMonsterID = 0x42

// PC98SwitchSilence 是換曲之間的靜音：`GAME.EXE $6509` `mov ax,320h ; call Delay`。
//
// 原版在這 800 毫秒裡**整個遊戲停住**（`Delay` 是空轉迴圈）；remake 只延後開播，
// 畫面照常走。
const PC98SwitchSilence = 800 * time.Millisecond

// PC98AreaSongs 是 `GAME.EXE $63CC` 的區域配樂表：ECL 區塊 → 曲號。
//
// 逐條讀自 `$6403..$64D8` 的比較鏈（`mov al,[9D3Fh]` 之後一串 `cmp ax,N ; jz`），
// 29 個區塊正好是 0..11 與 13..29，跳過不存在的 12（spec 101）。
// 表外的值（例如開機時的 `FFh`）直接返回，不派曲。
func PC98AreaSongs() map[int]int {
	table := map[int]int{}
	for song, blocks := range map[int][]int{
		2:  {0, 8, 11},
		3:  {2, 15, 18, 20, 29},
		4:  {14, 21, 24},
		5:  {19},
		6:  {1, 13, 16, 17, 28},
		7:  {22, 23},
		8:  {10},
		9:  {3, 4, 5, 6, 9},
		10: {7},
		13: {25, 26, 27},
	} {
		for _, block := range blocks {
			table[block] = song
		}
	}
	return table
}

// PC98AreaSong 回傳某個 ECL 區塊的曲號；表外的區塊回傳 false（不派曲）。
func PC98AreaSong(block int) (int, bool) {
	song, ok := PC98AreaSongs()[block]
	return song, ok
}

// PC98StartBlock 是還沒進任何區塊時 `[9D3Fh]` 的值。
//
// 開機初始化（`GAME.EXE $1BE4` 那一支）先把它設成 `FFh`（`$1C0B`），同一支的
// 結尾又設成 0（`$2019`）——所以開始選單與建角時的區域配樂查的是區塊 0，
// 放第 2 首。標題那一首只放到開始選單出現為止。
const PC98StartBlock = 0

// Scene 是派曲要看的畫面狀態，由遊戲每一影格填好交給 [Player.Update]。
type Scene struct {
	// Title 是標題畫面。
	Title bool
	// Adventure 是在地圖上（Amiga 的「冒險」情境用）。
	Adventure bool
	// Combat 是戰術盤面開著；BossCombat 是最後載入的怪物記錄為 [PC98BossMonsterID]。
	Combat, BossCombat bool
	// Shop、Temple 對應原版模式 1 的兩種服務。
	Shop, Temple bool
	// Ending 是結局。
	Ending bool
	// Block 是目前的 ECL 區塊；-1 代表不知道（不派區域配樂）。
	Block int
}

// sceneKind 是 PC-98 派曲在意的畫面種類。
type sceneKind int

const (
	kindNone sceneKind = iota
	kindTitle
	kindArea
	kindCombat
	kindShop
	kindTemple
	kindEnding
)

// pc98Kind 把畫面分類。順序照「哪一個模式會擋掉區域配樂」：
// 結局（模式 7）、戰鬥（模式 5）、商店與神殿（模式 1）都不走區域表。
func pc98Kind(scene Scene) sceneKind {
	switch {
	case scene.Ending:
		return kindEnding
	case scene.Combat:
		return kindCombat
	case scene.Shop:
		return kindShop
	case scene.Temple:
		return kindTemple
	case scene.Title:
		return kindTitle
	default:
		return kindArea
	}
}

// amigaCue 是同一個畫面在 Amiga 來源下的情境（spec 128，行為與先前相同）。
func amigaCue(scene Scene) Cue {
	switch {
	case scene.Combat:
		return CueCombat
	case scene.Adventure:
		return CueAdventure
	default:
		return CueTitle
	}
}
