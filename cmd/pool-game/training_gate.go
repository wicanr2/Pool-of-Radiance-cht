package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// `T)RAIN` 什麼時候出現在隊伍管理畫面（spec 008／097）。
//
// 原版 overlay-16 的隊伍管理主迴圈（`014Eh`）每一圈把十一個指令的啟用旗標
// 重設一次。旗標是選單記錄的 `+29h`，記錄在 `DS:062Dh` 起、每筆 `2Ah` 位元組
// （`022Fh` 讀 `[di+656h]` 判可不可按、`0256h` 讀 `[di+62Eh]` 拿標題的第一個
// 字母當快捷鍵）：
//
//	索引  0    1    2    3    4    5    6    7    8    9    10
//	旗標  656h 680h 6AAh 6D4h 6FEh 728h 752h 77Ch 7A6h 7D0h 7FAh
//	指令  C    D    M    T    V    A    R    L    S    B    E
//
//	019B  ds:5CF0h/5CF2h（目前這個角色）是 0 → 01EEh 全關、L 開
//	01B1  非 0 → 全開、L 關
//	01BF  **T 多一道**：es:[4937h]+550h > 0 或 ds:466Eh ≠ 0 才寫 1，
//	      兩個都不成立時**連寫都不寫**，保留上一次的值
//
// 開場隊伍是空的，走過 `01F8h` 把 `06D4h` 寫成 0，所以預設看不到 `T`
// ——spec 008 那兩張原版截圖都沒有它，因為兩張都不在訓練所裡。
//
// `[4937h]+550h` 依 spec 106 的 class 1 換算（`[4937h] + 2A00h + addr*2`，
// 要 mod 10000h）就是 **ECL 位址 `6DA8h`**：`2A00h + 6DA8h×2 = 10550h`。
// 同一條換算對得上 spec 100 的 `+5AAh ↔ 6DD5h`。
const trainingMaskAddress = 0x6DA8

// 全遊戲只有 ECL3 區塊 11 動 `6DA8h`（`tools/go.sh run ./cmd/pool-ecl-memory-audit
// -addresses 6DA8`），那個區塊就是訓練所／競技場。四道門各自把自己的職業
// 遮罩寫進 `@6E7C`，再由 `A007h` 的 `SAVE @6E7C, @6DA8` 抄進來：
//
//	9B9A  71h  MAGIC USERS
//	9B6D  72h  CLERICS
//	9FCF  74h  THIEVES
//	9FA1  78h  FIGHTERS
//
// 低四位是一位一類，高位 `70h` 四道門都一樣。訓練所自己再用
// `and ax, es:[di+550h]`（overlay-16 `2C95h`／`2CC3h`／`2CF6h`）逐人判職業，
// 隊伍畫面只看它非不非零。
const (
	TrainingMaskMagicUser = 0x71
	TrainingMaskCleric    = 0x72
	TrainingMaskThief     = 0x74
	TrainingMaskFighter   = 0x78
)

// stingPassword 是原版的除錯碼。隊伍管理畫面按 `J`（overlay-16 `049Ah`）
// 會讀一行字，和 `cs:0130h` 的 `STING` 比（`05BB:0724h` 是字串比較）；
// 對了就 `ds:466Eh = 1` 並印 `cs:0136h`。
//
// `466Eh` 全遊戲只有那一處寫，另有七處讀——`01C7h` 讓 `T` 無條件出現，
// 其餘六處（`29AAh`／`29E4h`／`2BAAh`／`2C83h`／`2CE4h`／`2E94h`）在訓練所
// 裡略過各自的閘門，`2BAAh` 就是 spec 097 的第二道。
//
// 主迴圈進來時 `0162h` 會把 `466Eh` 清成 0，所以它只在這一次進出裡有效。
const (
	stingPassword     = "STING"
	stingAcknowledge  = "I Understand, master..."
	stingMaxLength    = 16
)

// trainingHallOpen 是原版 `01BFh` 那一道：這一區的訓練所遮罩非零，
// 或者這一次進來輸入過除錯碼。
func (a *app) trainingHallOpen() bool {
	if a.stingUnlocked {
		return true
	}
	if a.eventMachine == nil {
		return false
	}
	return a.eventMachine.Memory[trainingMaskAddress] != 0
}

// stingInput 走 `J` 那一支的輸入。回傳 true 代表這一格按鍵被它吃掉。
func (a *app) stingInput() bool {
	if !a.stingPrompt {
		return false
	}
	if a.justPressed(ebiten.KeyEscape) {
		a.stingPrompt, a.stingBuffer = false, ""
		return true
	}
	if a.justPressed(ebiten.KeyBackspace) && len(a.stingBuffer) > 0 {
		a.stingBuffer = a.stingBuffer[:len(a.stingBuffer)-1]
	}
	for _, entered := range a.inputChars() {
		if entered >= 0x20 && entered <= 0x7E && len(a.stingBuffer) < stingMaxLength {
			a.stingBuffer += strings.ToUpper(string(entered))
		}
	}
	if a.justPressed(ebiten.KeyEnter) {
		// 原版比的是整串，不是前綴。
		if a.stingBuffer == stingPassword {
			a.stingUnlocked = true
			a.statusLine = stingAcknowledge
		} else {
			a.statusLine = ""
		}
		a.stingPrompt, a.stingBuffer = false, ""
	}
	return true
}
