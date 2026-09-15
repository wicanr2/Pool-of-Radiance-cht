# 主線連續實跑紀錄（issue #5）

測試：`tools/go.sh test ./cmd/pool-game -run '^TestMainlineProbeNaturalPartyFirstBattle$' -count=1 -v`
（`cmd/pool-game/mainline_probe_test.go`，路線規格 spec 137）。
隊伍：正常建角六人（`A`..`F`，預設職業與擲骰，HP 4／10／12／6／6／7），
`diceRoller` seed 136、`eclSeed` 1；沒有 direct-entry、旗標注入、forced-win 或強化。

## 2026-09-15：路線走通了，收據卻是假的

把第 6～11 段（東航線 → 野外 → 波多廣場 → 斯托亞諾夫城門 → 城堡內部繞四張圖
→ 覲見廳 → 結局）改成明確的狀態機之後，整條測試從標題按到結局過場**通過**，
`4ABA = FEh`、三頁結局、回到 `ECL3/0`，全程約 1 秒。

但戰鬥記錄印出來是這樣的：覲見廳第一場（12 名 8TH LVL FIGHTER）第 5 回合六個人
**全部倒地**（狀態 5），敵方「FOUND NO TARGET」；緊接著第二場泰倫斯拉克斯，
六個人**滿血站著**。原因有兩個，都是 remake 的缺陷，這一輪一起修了：

| 缺陷 | 位置 | 修法 |
|---|---|---|
| 戰鬥的生命值是另一份陣列，打完從來不寫回隊伍；倒地與死亡也不寫回。每一場打完都是滿血 | `finishCombat` | `storeCombatHitPoints`：寫回 `+11Bh`／`+10Ch`，狀態換算照 overlay-05 `04ADh` `0636h..0686h`（打贏時 3→0、5→4、4 且有 HP→0）；昏迷／倒地／死亡的人不再上戰場 |
| 全滅之後停在 COMBAT 邊界的 pending 還留著，任一個 ENTER 就把戰後腳本當打贏往下跑；結果碼 `6DC7` 從來沒寫過，腳本看到的永遠是「贏」 | `finishCombat` | 打贏 `6DC7 = 0`（overlay-05 `14CAh` 進門就清）；全滅 `6DC7 = 80h`，印 `The END!`／`The monsters rejoice for the party has been destroyed`／`Press any key to continue`（overlay-05 `1471h`／`147Ah`／`14B0h`），等一鍵回標題（`4961h` 的消費者未讀，回標題是 hypothesis） |

修完之後同一條測試的結果：**貧民窟第 4 場之後全滅**（`4ABB = 04`），畫面停在
`The END!`。這才是這支隊伍、這種走法（打完不休息、不治療、不訓練）在原版規則下的
真實結果。先前 issue 留言裡「已自然完成貧民窟 25 場、索寇要塞、City Hall 交件」的
收據，全部是在「生命值不會掉」的 remake 上量到的，不能算數。

## 走通的段落（機制層面，用注入的 60 HP 隊伍或壞掉的 HP 量到）

| 段 | 證據 | 備註 |
|---|---|---|
| 港務長 EAST → 碼頭 → 野外 27 (9,29) → 跨到 26 → (11,28) NORTH → `ECL1/18` | 本測試 log `landed in the wilderness`／`entered Podol Plaza` | 與 `enterStojanowGate` 同一條路 |
| GEO1/18 (4,0) 朝北 → `ECL2/9` | `Podol north edge: block 18 → 9` | |
| 馬車（COMBAT 殺商人）→ 索賄付 15 金 → (8,5) → 北緣 → `ECL5/3` | `got the wagon`／`through the gate`／`gate north edge: block 9 → 3` | 索賄 `RANDOM 9` 擲 0 會「IMPOSTERS」，選 FLEE 回 (4,14) 再試即可（不寫旗標）|
| 城門格 (4,8) → `ECL5/5`；GEO3→6→5→4→3 繞一圈到 (13,15) 朝東 → `ECL5/7` | `TestCastleStairsLeadToTheAudienceHall`（0.3 s）與 `TestTheCastleBehindStojanowGateHasContent` | 上樓落點腳本寫 (5,7)，remake 再加一步成 (6,7)（hypothesis） |
| (3,8) 覲見廳 → 衛兵 → 投票 ATTACK → 泰倫斯拉克斯 → `PROGRAM 8` → 回 `ECL3/0` | 同上兩條 | |

另外修掉一個走一步入口的缺陷：印完字之後才寫的座標（`SAVE → C04B/C04C` 接
`NEWECL`）以前會被丟掉（`EXIT` 不帶事件的那一份 result 沒有套用），上樓落點因此留在
原格；`moveInitialDungeonForward`／`runBlockedInitialCellEntry` 現在把它收下。

## 下一步（尚未做）

測試駕駛缺的是**玩家策略層**：打完看 HP 決定要不要紮營休息（每 24 小時回 1 點）
或去神殿治療、有 XP 就去訓練所升級、拿到錢就買裝備。沒有這一層，自然隊伍在
貧民窟就會全滅；有了這一層，城堡那兩場（12 名 8 級戰士加一條龍）需要的等級
也還要靠其他委任累積。這一段的規模比原本 #5 估的大得多，先回報再決定。
