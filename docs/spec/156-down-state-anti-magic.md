# Spec 156：倒下的狀態依打穿點數、84h 倒下、反魔法區擋施法、戰場上解除效果

狀態：READY（下表每一條讀過位元組並實作，從 `Update()` 送鍵測試）。證據是位元組（exact），
沒有原版執行期收據逐次對過。日期：2026-09-27。主台帳：GitHub issue #115、#116、#117。
接手 spec 155〈與原版不同〉的「倒下一律寫 5」與 spec 149〈未閉合〉的 84h 兩條。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | `0051h` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 15 | `470de2bf7caeb6bfccfdcff831f3713d64fea730fc04845c541f5bca36e3cb08` | `00ACh` |
| 19 | `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9` | `00C9h` |
| 24 | `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714` | `0100h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |

- 反組譯：`coab-go-test:20260729` 的 GNU objdump 2.40（`-D -b binary -m i8086 -M intel`）。far call 的
  segment 以 (`executable_file_offset` − stub offset − 3B0h) ÷ 16 反查；overlay-25 entry 28 的 stub 在
  `0ACh`（`20h + 28 × 5`），呼叫形狀是 `9A AC 00 0A 01`。

## 倒下的狀態：overlay-25 entry 28（`2266h`，exact）

```
2277..2297  +11Bh ≥ 傷害 → [bp-2] = 剩餘；否則 [bp-1] = 傷害 − +11Bh（打穿的點數）
22AF  80 7E FF 09 / 77 11           打穿 > 9 → 22C6h
22B5  80 7E FE 00 / 75 16           剩餘非 0 → 22D1h
22BE  26 80 BD 0C 01 01 / 75 0B     剩餘 0 而 +10Ch 是 1 → 22C6h
22C9  26 C6 85 0C 01 06             +10Ch = 6
22D1  80 7E FF 00 / 76 21           打穿 1..9：
22DA  26 C6 85 0C 01 05               +10Ch = 5
22E0  80 3E 54 49 05 / 75 0F          戰鬥中（DS:4954h == 5）→
22E7  8A 46 FF / … / 26 88 45 0E       +108h 的 +0Eh（倒地計數）= 打穿的點數
2301  26 C6 85 0C 01 04             剛好 0 點 → +10Ch = 4
```

`gamepack.ApplyDamage`（`internal/gamepack/damage.go`）早就照這一段寫；缺的是戰鬥裡的呼叫端沒用它。
這一支的三個呼叫端（全 38 顆 overlay 掃 `9A AC 00 0A 01`）：

| 呼叫端 | 是誰 |
|---|---|
| overlay-13 `048Dh` | 近戰（entry 4 `02FEh`）|
| overlay-24 `14FBh` | 法術與 84h 的傷害（entry 19 `133Ah`）|
| overlay-03 `2A71h` | ECL 的扣血（戰鬥外，remake 已經走 `ApplyDamage`）|

倒地計數在回合收尾（overlay-08 `08C6h` `26 FE 45 0E` 加一、`08D2h` `26 80 7D 0E 09` 大於 9 轉 6）接著數，
所以打穿 k 點（1..9）的人撐 10 − k 個回合收尾，不是一律十回合。

remake：`tacticalState.settleDownState`（`cmd/pool-game/death_effects.go`）把 `ApplyDamage(0, 狀態, 打穿)`
的狀態寫回、狀態 5 時把 `DyingCounters` 設成打穿的點數。近戰（`resolveAttackSwings` 倒下那一段）、
法術（`applySpellDamage`）、84h（`woundWearer`）都走它；毒死（`005Ah`）本來就直接寫 6。

## 84h 在自己的物品選單裡倒下（exact）

- **群組 13**：overlay-24 entry 19 的 `1610h..161Dh` 就是 `combatantDown`（spec 155）。`alignedWear` 戰鬥中
  倒下那一段原本自己摘 `EscapeStrippedEffects`，改成把手上的串列放回盤面、叫 `combatantDown(state, 那一格,
  打穿點數)`、再拿回來（`wearItem` 最後把它寫回盤面）。屍體表、十六個碼、群組 13 與其他倒下入口同一支。
- **選單不收**：overlay-19 entry 6 的迴圈（`0F2Fh`）只在按鍵落在 `0E66h` 那一組、結果非 0（`0F42h`
  `26 80 3D 00`）或身上沒東西（`0F51h` `26 80 BD C7 00 00`）時離開，倒下不在其中。
- **Use 不接**：`0F8Fh` `26 80 BD 0D 01 00 / 74 5F`——`+10Dh` 為 0 就不組 " Use"。remake 的
  `combatItemUseOpen` 加上「體型不為 0」，選項列與 U 鍵都走它。
- **回合怎麼收**：overlay-08 玩家回合（`0307h`）只在進來時看一次 `+10Dh`（`0310h` `26 80 BD 0D 01 00 / 75 03`，
  為 0 就 `05CDh` entry 34 結束）。'U'（`03E9h` `3C 55`）叫 overlay-19 entry 6（`03F2h` `9A 3E 00 C9 00`）、
  overlay-13 entry 8（`03FDh`）、結果為 0 再 overlay-25 entry 41（`0408h`），然後 `05A6h`：結果 0 → 重畫
  （`05B8h` ov32 entry 22、`05C3h` ov25 entry 5）→ `036Ch` 回到讀指令（`0381h` `E8 CA 02` → `064Eh`）。
  整條迴圈與 `064Eh` 都沒有再看 `+10Dh`／`+10Ch`，所以 ESC 之後輪到的還是那一位，指令列照常。
  remake 原本就是這個流程（`combatItemInput` 的 ESC 只收選單），測試把它釘住。

## 反魔法區：`[4933h]+1CAh`（ECL `@49E5`，exact）

class 0 的窗：`6E00h + 49E5h × 2 ≡ 1CAh (mod 10000h)`。唯一的腳本寫入端是 ECL8 block 16 `9BEEh` 的 SAVE
（`cmd/pool-ecl-memory-audit -addresses 49E5`），區塊載入時 overlay-07 `025Ah` 清成 0（`block_load.go`）。
全 38 顆 overlay 掃 `26 83 BD CA 01 00`（`cmp word es:[di+1CAh], 0`，前一條都是 `C4 3E 33 49`）：

| 位址 | 做什麼 | remake |
|---|---|---|
| overlay-08 `073Ah` `75 28` | 戰鬥指令列的 `Cast ` 不組 | `combatSegmentShown`（Cast）與 `openCastMenu`（C 鍵）|
| overlay-15 `0372h` `76 16` | entry 8（`035Dh`）模式 1：印 `02F3h` "cannot cast spells in this area"（`045Eh` 帶 `DS:5CF0h` 的名字）、回 0；entry 2（`0497h`，營地／冒險的施法）在 `04B2h` 以模式 1 叫它，回 0 就離開 | `fieldCastAdvance` 挑人那一步印 `msgFieldCastAntiMagic`、不列法術 |
| overlay-19 `0F9Bh` | 物品選單的 " Use" | 既有（`outsideAntiMagic`，spec 149）|
| overlay-09 `0435h` `74 03` | AI 用物品（entry 3）不用 | 未接（`foe_items.go`，不在本 spec 範圍）|

entry 8 的模式 2、3（`0903h`、`0B57h` 的兩個呼叫端）不過這一道：只有施法擋。AI 施法（overlay-09 entry 4）
不讀 `1CAh`，怪物在反魔法區照樣施法。

## 戰場上解除效果改盤面那一份（#116）

原版的效果串列只有一份，掛在記錄 `+7Fh`（spec 069）。remake 戰鬥中那一份是 `tacticalState.Effects`，
收場時 `storeCombatEffects` 用它蓋回隊伍。`castSpell` 的 `RemoveEffects` 那一支（解盲、解咒、解病的
`225Bh`）原本改的是 `a.state.Party[].Effects`，收場就被蓋回去，等於白解。改成改 `state.Effects[目標]`
（同時寫一份回隊伍，與力量那一支相同），怪物目標也照樣解。

盤點戰鬥中改 `Party[].Effects` 的其他路徑：`wearItem`（戰鬥中改盤面那一份再寫回）、力量那一支
（同上）、`field_cast*.go`／`poison.go` 的 `slowPoisonAftermathInCamp`／`advancePartyEffects`／`race_effects.go`／
`addnpc.go`（都只在戰鬥外），沒有第二處白解。

## 測試

`cmd/pool-game/down_state_test.go`、`combat_cure_test.go`，從 `Update()` 送鍵：

- `TestDownStateFollowsTheOverkill`：近戰打穿 0／3／9／10 點與狀態 1 打穿 0 點 → 4／5／5／6／6，倒地計數
  3／9；魔法飛彈打穿 0／2 點 → 4／5。
- `TestDyingCounterStartsAtTheOverkill`：打穿 7 點，三個回合收尾後死透。
- `TestAlignedSwordDownInOwnItemMenu`：84h 打穿 5 點（狀態 5、計數 5）與 10 點（狀態 6），屍體表記下、
  選單還開著、選項列沒有 USE、U 不做事、ESC 後輪到的還是他。
- `TestAntiMagicShellBlocksCasting`：`@49E5` 非 0 時 Cast 不在指令列、C 不開清單；營地挑人之後印
  "cannot cast spells in this area"；清掉之後都恢復。
- `TestCureDiseaseOnTheBoardStaysCuredAfterCombat`：戰場上解病（解病術參數表模式 0，打自己），盤面
  那一份沒了，`storeCombatEffects` 之後隊伍那一份也沒了。

## 與原版不同、寫明的幾處

| 原版 | remake | 理由 |
|---|---|---|
| 營地的反魔法訊息以 entry 20(記錄, 字串, 0Ah, 1) 印在底列並停一拍 | 印在施法頁的訊息那一行 | 呈現（停止線）|
| 倒下的人在指令迴圈裡還能下 Move／Aim 等指令，各指令對體型 0 的行動者做什麼沒有讀 | 照 remake 各指令現有的處理 | 只有 84h 在自己的選單裡倒下才走得到；停止線 |

## 被推翻的斷言

| 舊寫法 | 現況 | 依據 |
|---|---|---|
| `[4933h]+1CAh` 只有 overlay-07 `025Bh` 寫 0，四個閘門在這一版永遠不成立（spec 096〈entry 3〉與〈還沒讀〉、`combat_screen.go`、`foe_items.go` 的註解）| ECL8 block 16 `9BEEh` 的 SAVE 寫 `@49E5`；overlay 的 disp16 掃描結構上看不到 ECL 的寫入 | `cmd/pool-ecl-memory-audit -addresses 49E5` |
| 輪到的人 `+10Dh` 對 Use 那一道恆成立（`item_page_trade.go` 舊註解）| 84h 在自己的選單裡倒下之後不成立 | overlay-19 `0F8Fh`、`0F42h`、`0F51h` |
| 近戰與法術倒下一律寫狀態 5（spec 155〈與原版不同〉）| entry 28 分 4／5／6，5 時倒地計數是打穿的點數 | overlay-25 `22AFh..2301h` |
