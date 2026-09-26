# Spec 145：建角依種族掛的效果

狀態：CONFORMED（掛哪些碼、節點內容、順序、舊存檔補上；`cmd/pool-game/race_effects_test.go`
從 `Update()` 建六個種族，逐位元組對原版 overlay-16）；各碼在規則裡的作用見 spec 112。
日期：2026-09-26。主台帳：GitHub issue #88（#86 讀群組 9／10／16 時發現）。

## 一句話

原版建角選完種族就依記錄 `+2Eh` 掛上永久、解除魔法解不掉的效果節點：矮人
`5Ah 61h 1Ah 2Fh`、侏儒 `61h 12h 2Fh 30h`、精靈 `6Bh`、半精靈 `7Ch`、半身人 `5Ah 61h`，
人類不掛。每個節點都是 `碼 00 00 FF 00`。remake 在 #88 之前沒掛，所以矮人與侏儒對類人的
+1、侏儒被 BUGBEAR／GNOLL 打的 −4、精靈與半精靈抗睡眠魅惑，都只有匯入的原版角色才有。

## 輸入

- overlay-16：SHA-256 `a142d8a8f3b3c46a7755cf231228e7981c5b9f77721313ce79ab6c0aec105b94`
  （`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）。
- overlay-24：`e878166e…`（同一份清冊）。
- 工具：`objdump -D -b binary -m i8086 -M intel`（`coab-go-test:20260729`），位址是 overlay
  code 內的位移。
- 原版預設人物：`Pool of Radiance (1988).zip` 的 `poolrad/chrdat*.sav`／`.spc`。

## overlay-16 `0772h..0905h`（exact）

建角的種族選單回來之後：

```
0761  call 011Dh:0039h            ; 選單；[bp-12h] = 選到的列
0778  cmp [bp-12h], 6 / jne / inc ; 第 6 列之後跳一格（半獸人不開放），所以人類是 7
0790  26 88 45 2E                 ; 記錄 +2Eh = 種族
0797  26 8A 45 2E                 ; 讀回來分支
079B  3C 05 75 3A                 ; 5 半身人：+0C0h = 1；掛 5Ah、61h
07D9  3C 01 75 68                 ; 1 矮人：  +0C0h = 1；掛 5Ah、61h、1Ah、2Fh
0845  3C 03 75 67                 ; 3 侏儒：  +0C0h = 1；掛 61h、12h、2Fh、30h
08B0  3C 02 75 22                 ; 2 精靈：  +0C0h = 2；掛 6Bh
08D6  3C 04 75 22                 ; 4 半精靈：+0C0h = 2；掛 7Ch
08FC  26 C6 85 C0 00 02           ; 其他（人類）：+0C0h = 2，不掛
0905  （會合，接著建角的下一步）
```

每一次掛都是同一段 17 bytes（以矮人第一個 `07A8h` 為例）：

```
07A8  FF 76 FE / FF 76 FC         ; push 記錄遠指標
07AE  B0 5A 50                    ; push 碼
07B1  31 C0 50                    ; push 0（持續）
07B4  B0 FF 50                    ; push FFh
07B7  B0 00 50                    ; push 0
07BA  9A 52 00 00 01              ; call 0100h:0052h = overlay-24 entry 10（spec 109）
```

overlay-24 entry 10（`0E54h`，`retf 0Ch`）把參數寫進新節點（`0ED3h..0EF6h`）：
`[bp+0Ch]` → `+0`（碼）、`[bp+0Ah]` → `+1..+2`（持續）、`[bp+06h]` → `+4`、
`[bp+08h]` → `+3`。所以節點是 `碼 00 00 FF 00`：

| 欄 | 值 | 意思 |
|---|---|---|
| `+1..+2` 持續 | 0 | 永久：overlay-20 `0165h` 不遞減不到期，有限持續的新效果取代不了（spec 069）|
| `+3` | `FFh` | 解除魔法跳過（spec 098 `23BEh`：`+3 >= FFh`）|
| `+4` | 0 | 摘掉時不叫處理常式 |

`+0C0h` 是圖示大小（`internal/character/dos.go` 的 `offsetIconSize`），小號是矮人、侏儒、
半身人——與 remake 既有的 `creation.SmallRaceIDs` 一致，當作這一段讀對了的正對照。

**順序**：entry 10 一律接在串列尾端（spec 069），而這是建角那一刻，所以種族節點是
串列最前面那幾個（strong inference：沒有讀到建角開頭清空 `+7Fh` 的那一行，但新記錄在
這之前沒有任何掛效果的路徑）。

## 原版預設人物（反例）

二十份 `chrdat*.sav` 的 `+2Eh` 全是 7（人類），`+0C0h` 全是 2；七份 `.spc` 裡只有
`3Dh`、`26h`、`59h`（裝備給的，spec 069），沒有任何一個種族碼。與「人類不掛」一致；
正面的位元組對照只能取 overlay-16 本身。

## 各碼做什麼（spec 112）

| 碼 | 群組 | 作用 | remake |
|---|---|---|---|
| `12h` | 10 | 侏儒打 KOBOLD、GOBLIN 等四種 +1 | `HitRollEffects` |
| `1Ah` | 10 | 矮人打 ORC、HOBGOBLIN 等八格 +1 | `HitRollEffects` |
| `2Fh` | 16 | OGRE、TROLL、巨人一類 −4（比對的名字與記錄見 spec 112 的表）| 未接（spec 112〈OPEN〉）|
| `30h` | 16 | 被 BUGBEAR、GNOLL 打 −4 | `HitRollEffects` |
| `6Bh` | 9 | 1d100 <= 90 擋睡眠、魅惑 | `SpellEffectImmunity` |
| `7Ch` | 9 | 1d100 <= 30 擋睡眠、魅惑 | `SpellEffectImmunity` |
| `5Ah`／`61h` | 12 | 依體質（`+14h`）加豁免：`5Ah` 只在類別 0（毒），`61h` 在類別 2（魔杖）與 4（法術）；4..6 +1、7..10 +2、11..13 +3、14..17 +4、18..20 +5（spec 112〈群組 12／6／4／5〉，#96）| `SaveRollEffects`；送鍵測試 `TestDwarfOutsavesAHumanOnTheSameRoll` |

## remake

| 原版 | remake |
|---|---|
| `078Ah..0905h` 的分支表 | `gamepack.RaceEffectCodes`（照推入順序）|
| entry 10 的參數 `(碼, 0, FFh, 0)` | `gamepack.RaceEffectNode` = `NewEffectNode(碼, 0, EffectUndispellable, false)` |
| 建角寫記錄 | `(*app).finishCharacter` 的 `Effects: raceEffectNodes(種族)`；`.SPC` 匯出跟著帶 |
| — | 讀檔補上：`(*app).migrateRaceEffects`，與 #74 的 `migrateNPCMorale` 同一個時機（`restoreCampaign`），沒有戰役的讀檔（`loadSavedGame`）也叫。隊伍與人物名單都補；缺的碼補在串列最前面，已經有的不重掛；NPC 不動 |

測試（`cmd/pool-game/race_effects_test.go`）：

- `TestCreatingEachRaceAttachesTheOverlay16Effects`：從 ZIP 讀 overlay-16，掃 `3C rr 75` 與
  上面那 17 bytes 得出每個種族的節點當 oracle（先驗 `0790h` 的 `26 88 45 2E` 當位址基準的
  正對照），再從 `Update()` 建六個種族，隊伍裡那一位的 `.SPC` 逐位元組相等。
- `TestLoadingAnOldSaveAddsTheRaceEffects`：#88 之前形狀的存檔（侏儒身上已有詛咒），讀兩次；
  種族節點補在詛咒前面、只補一份，人類與 NPC 不動，人物名單也補。
- `TestCreatedGnomeIsHarderForABugbearToHit`：`Update()` 建的侏儒帶著自己的串列進盤面，
  BUGBEAR 同一骰（13）打不中，不帶的對照組打中。

## 未閉合

- `2Fh` 仍無 remake 消費端（spec 112〈OPEN〉）；節點已經掛上，接上規則時不必再動建角。
- 串列在建角開頭是空的，是 strong inference（見〈順序〉）。
