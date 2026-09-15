# Spec 137：主線必經路徑——從標題到結局要走過哪些區塊

狀態：READY（各段的區塊、旗標、入口與出口都對回原版 ECL trace 或既有 spec；
第 6～11 段在同一條測試裡以正常按鍵接續走通）；DRAFT（自然強度的隊伍走不到——
見「撐強度的段落」；上樓落點是否再加一步；全滅後回標題）。
日期：2026-09-15。主台帳：GitHub issue #5。

## 一句話

結局只掛在一個旗標上：`ECL5/7` 打贏泰倫斯拉克斯後 `A815h SAVE FEh → 4ABA`，
接著 `PROGRAM 8` 播結局。走到那一格**不需要任何委任**——`4AC1`（公告進度）
在全部 29 個區塊裡只有 `ECL3/0`（挑公告）與 `ECL3/8 A592h`（墓園委任要 ≥ 4）
讀它，城堡四個區塊一個都不讀。所以「必經」只有一條：進波多廣場、出北緣過
斯托亞諾夫城門、進城堡、上樓、站上覲見廳那一格。委任是隊伍強度的來源，不是
腳本上的閘門。

## 證據

- `ECL5/7` 全區塊 trace：`tools/go.sh run ./cmd/pool-ecl-trace -archive 5 -block 7`
  （29／29 區塊靜態可走，見 WORKLIST「Pool ECL record format 閉合」）。
- `4AC1` 的引用清冊：`docs/audit/dos-ecl-campaign-memory-refs.json`（`exact`；
  三個曾經解不開的區塊現在也走得完，所以這是全集不是下界）。
- 城門與城堡：spec 101「區塊 6：斯托亞諾夫城門怎麼過」與「城堡裡面的出邊」。
- 碼頭與航線：spec 102；野外落點與位移：spec 105。
- 委任槽：spec 041。

## 必經段落（依序）

「入口條件」是進這一段前記憶體要長什麼樣；「出口」是這一段結束時的可判訊號。
證據等級標在每列。

| # | 段落 | 區塊／圖 | 入口條件 | 怎麼走 | 出口訊號 | 等級 |
|--:|---|---|---|---|---|---|
| 1 | 建 6 人隊伍、開場導覽 | 標題→`ECL3/0` | 全新遊戲 | 正常建角 6 次、`B`egin、導覽全按 ENTER | `introDone`、隊伍 6 人 | exact（既有測試）|
| 2 | 貧民窟：首戰 + 25 場 | `ECL2/20`／GEO2/20 | 站在城區 | 城區 (0,4) 往西出界；區內巡邏打遭遇，`B69Ch` 每場 +1 | `4ABB == FEh` | exact（spec 042）|
| 3 | 市政廳交件 + 存讀檔 | `ECL3/8` | `4ABB == FEh`、`4A01 == 0` | 貧民窟 (15,4) 往東回城，(3,4) 東進市政廳到職員 (5,5)；F10、回標題、L | `4ABB == FFh`、`4AC1 == 1` | exact（spec 041）|
| 4 | 索寇要塞 | `ECL4/21` | `4AA7 < FEh` | 港務長 (11,2) 朝北：第一次免費給索寇票（`4A01 = 1`）；(15,1) 上船；對費瑞先「說謊」（`4A01 = FFh`）再了結亡魂（`4AA7 = FEh`、`4A26 = FFh`）；回程船 | `4AA7 == FEh` | exact（spec 102；順序限制見 CONTEXT「碼頭那條主線」）|
| 5 | 市政廳交索寇件 | `ECL3/8` | `4AA7 == FEh` | 同 #3 | `4AA7 == FFh`、`4A01 == 0`（adapter，strong inference，見 `applySokalHandInTicketState`）、**`4A00 == 0`**（職員 `9C3Ah` 每次清） | exact／strong inference |
| 6 | 東航線進野外 | `ECL8/27`→`ECL7/26` | `4AA7 ≥ FEh`、`4A01 == 0`、有白金 | 港務長選 EAST（`4AC4 = 1`），(15,1) 上船，落在圖 27 (9,29)；往西跨到圖 26，第二步踩 (11,28) 選 NORTH | 進 `ECL1/18` | exact（spec 105 實測；`enterStojanowGate` 走的就是這條）|
| 7 | 波多廣場北緣 | `ECL1/18`／GEO1/18 | 站在 18 | 走到 (4,0) 或 (11,0) 朝北踏出（`99D4 ON GOTO @C04D` 第 0 支：`NEWECL 9`） | 進 `ECL2/9`，落點 (4,15)／(11,15) | exact（`TestTheNorthEdgeOfBlockEighteenLeadsToBlockNine`）|
| 8 | 斯托亞諾夫城門 | `ECL2/9`／GEO2/9 | **`49C9 < 14`（時刻）、`4A00 == 0`、`4A77 & 64 == 0`**；避開地形 3（衛兵，踩到 `4A78 → 3` 之後索賄必翻臉） | 地形 9 的 (4,14)／(11,14) 遇馬車商人：COMBAT 殺了搶走或 PARLAY 花 250 金（都 `B014 OR @4A77 64`）；地形 5 付 15 金，腳本把隊伍搬到 (8,5)；北緣 (4,0)／(11,0) 朝北踏出 | 進 `ECL5/6`→立刻 `NEWECL 3`，停在 `ECL5/3` GEO5/3 | exact（spec 101；`TestPayingTheTollAtStojanowGateOpensTheCastle`）|
| 9 | 進城堡內部腳本 | `ECL5/3`→`ECL5/5`，GEO5/3 | 站在院子（落點 (4,15)） | 走到城門格 (4,7)／(4,8)（地形 21；`ecl5/3 9B5Dh` 表索引 17 → `B38Ah NEWECL 5`）。區塊 5 **不換圖**，仍在 GEO5/3 | `ECL5/5`，GEO5/3 | exact（trace ＋ `TestCastleStairsLeadToTheAudienceHall`）|
| 10 | 內部繞四張圖到樓梯 | `ECL5/5`，GEO5/3→6→5→4→3 | 站在區塊 5 | 只走地形 0：GEO3 南緣 (4,15) → GEO6 → 東緣 (15,11) → GEO5 → 北緣 (4,0) → GEO4 → 西緣 (0,11) → GEO3 (15,11) → (13,15)（134 步，`workplace/castleq`）。換圖表在 `ecl5/5 9AEBh`（`castleInteriorExits`），靠 `49C5` 選表。(13,15) 地形 4，朝東踏 → `9B28h` → `A467h`「YOU GO UPSTAIRS.」→ `NEWECL 7` | `ECL5/7`，GEO5/7；落點腳本寫 (5,7)，remake 再加一步 → (6,7) | exact（路線與換圖表）；hypothesis（落點是否再加一步）|
| 11 | 覲見廳 | `ECL5/7`／GEO5/7 | 站在 GEO5/7；地形 8 只有 (3,8) 一格 | 走到 (3,8)：`A455h` 印覲見廳文字 `4A6D |= 8`；`A558h` 兩名衛兵遭遇選單，結果碼 1 = COMBAT（`LOAD MONSTER 85 ×12`），打贏 `4A6D |= 4`；`A629h` 龍的說詞，每個角色投票 `ATTACK`／`JOIN`（選 JOIN 的角色被龍附身離隊）；`A7DCh` 最後一戰 `LOAD MONSTER 42h`（TYRANITHRAXUS，HP 80 AC 0，2×2）| `4ABA == FEh`、`endingActive`、三頁結局、`NEWECL 0` 回城 | exact（trace；戰鬥本身 `TestDefeatingTyranthraxusSetsTheVictoryFlag`）|

城堡是四張 GEO 拼成的 32×32（換圖表 `9AEBh`：3 東→4、南→6；4 南→5、西→3；
5 北→4、西→6；6 北→3、東→5），`ecl5/3`／`5/4`／`5/6` 是外圈腳本，`ecl5/5` 是
內部腳本，靠地形碼互換：外圈踩地形 21／22 → `NEWECL 5`；內部踩地形 30／31 →
`NEWECL 3`（`ecl5/5 9C1Eh`）。所以 GEO5/3 上的 (1,3)（也是地形 4）在內部腳本裡
到不了——它被 (3,7)／(3,8) 的地形 30 隔開，而 (13,15) 才是繞一圈之後站得上去的那一格。

GEO5/7 的地形格（`workplace/openreach -block 7 -from 5,6`）：

```
 4  . . . . . 3 . . . . . . . . . .
 5  . . 7 . . . . . 4 . . . . . . .
 6  . . 6 9 A @ . . . . . . . . . .      @ = 落點 (5,6)
 7  . . . . . 1 . . . . . 5 . . . .
 8  . . . 8 . . . . . . . . . . . .      8 = 覲見廳 (3,8)
```

地形 4 是活板門（會有人石化）、5 是美杜莎、6／7 是賊與密室的 NPC——都不在
(5,6) → (3,8) 的路上，規劃器直接避開。

## 不是必經、但撐強度的段落

城堡兩場：12 名衛兵（`mon5/85`，8TH LVL FIGHTER）與泰倫斯拉克斯。打不打得過取決於
隊伍等級與裝備，不取決於旗標。照必經路徑跑一次的結果（2026-09-15，
`docs/playtest/mainline-end-to-end.md`）：正常建角的一級隊伍在戰後生命值會寫回的
規則下，**貧民窟第 4 場就全滅**；覲見廳那一場則是六人第 5 回合全部倒地。所以
「委任不是閘門」是腳本層的事實，而「不打委任、不休息、不訓練就到不了結局」是
規則層的事實，兩者同時成立。下一段要補的是測試駕駛的玩家策略層（休息／神殿／
訓練／裝備），不是改規則。

## 已知會把路走死的狀態

| 狀態 | 效果 | 誰寫的 | 對策 |
|---|---|---|---|
| `4A00 != 0` 進城門 | 馬車商人不出現，索賄那一支轉去打熊地精，城門永遠過不去 | 貧民窟／索寇要塞 `SAVE 255`、墓園 `ADD 1`、市政廳 reward 差值 | 市政廳職員 `9C3Ah` 每次進門清 0——交完索寇件就直接去碼頭，中間不進墓園 |
| `49C9 ≥ 14` 站上地形 9 | 商人 `ADAAh` 直接 EXIT | 遊戲時鐘 | 紮營休息到早上再踩 |
| `4A78 ≥ 3` | 索賄熊地精「THEY ARE IMPOSTERS!」 | 踩到地形 3 衛兵 | 規劃路線避開地形 3 |
| `4A01 != 0` 找港務長 | 不開口 | 票沒清 | #5 的 adapter；靠港前先確認 |
| 投票選 JOIN | 該角色 `6C0C = 81h` 離隊 | 玩家選項 | 每個角色都選 ATTACK |
| 內部腳本踩到地形 30／31 | `NEWECL 3` 回外圈，樓梯格從此到不了 | GEO5/3 (3,7)(3,8)、GEO5/5 (12,7) | 規劃只走地形 0 |
| 走一步的入口 0 印完字才寫座標、之後 `EXIT` 不帶事件 | remake 曾把那一份 `Writes` 丟掉：上樓落點留在原格再加一步（(14,15)）而不是腳本寫的 (5,7) | `moveInitialDungeonForward`／`runBlockedInitialCellEntry` 只在有呈現事件時套用 | 2026-09-15 修正：沒有事件的 `EXIT` 也套用 |

## 驗收

`TestMainlineProbeNaturalPartyFirstBattle`（`cmd/pool-game/mainline_probe_test.go`）
從標題按到結局，每一段是有界的狀態機步驟；撞 guard 就 `Fatalf` 印出
ECL／GEO／座標／朝向與 `4ABA 4AC1 4AA7 4A01 4A00 4A6D 4A77 49C9`。
人讀的經過紀錄在 `docs/playtest/mainline-end-to-end.md`。
