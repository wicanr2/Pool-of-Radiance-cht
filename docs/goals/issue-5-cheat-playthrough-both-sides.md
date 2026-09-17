# Goal：兩邊都作弊通關——remake 用 F6 作弊選單、原版用 dosgolem 鎖 HP，從標題跑到結局並逐段對照（GitHub #5）

主台帳：[issue #5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)（主線從開場到結局連續跑過一次）。
相關：[#22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（不開作弊的正常強度，改為可選）、
[#42](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/42)（休息後重複文字，對拍停在這裡）。
上一個 goal 的結案位置：[`issue-43-cheat-menu-norris.md`](issue-43-cheat-menu-norris.md)〈2026-09-17 收在哪〉。

## 驗收口徑（使用者 2026-09-17 決定，CLAUDE.md §3 已寫入）

- **算主線收據**：用產品的作弊選單（spec 141：F6、鎖 HP、一擊斃命）從標題以正常按鍵跑到結局。
- **原版那一側同樣作弊通關**。使用者的原話是「dos 也可以作弊通關喔，dosgolem 是我們控制的」：
  把 dosgolem 擴充成能在原版裡鎖 HP，跑同一條路線，兩邊逐段對照必經區塊與旗標。
- **不開作弊的正常強度通關**：改為可選（#22）。
- **仍然不能用的捷徑**：作弊選單以外的傳送、注入旗標／座標／時鐘／金錢／經驗值。

## 現況（2026-09-17，`04255a2` 之後）

**remake 那一側**：
- `TestMainlineProbeHPLockedRouteA`（house rule、seed 142、路線 (a)）已經從標題跑到結局（`4ABA = FE`、結局 3 頁，
  playtest 補十三）。鎖血用的是測試治具：`hp_lock_test.go` 掛在 `afterTick`，呼叫產品的
  `restorePartyHitPoints`。這條測試沒有開產品的作弊選單，也沒有一擊斃命。
- 產品的作弊選單已經有了（`cmd/pool-game/cheats.go`），駕駛有 `setCheats(lockHP, oneHitKill bool)`
  （`cheats_test.go`），`TestMainlineProbeHouseRuleCheatAtNorris` 只在諾里斯那一場開。
- 路線 (a) 的段落與旗標：spec 137〈路線 (a) 的 HP 鎖定診斷〉。依序是
  - 貧民窟 25 場（`4ABB`）
  - 交件（`4AC1`、`4AB0 = 1`）
  - 波多廣場拍賣（`4AB0 = FE → FF`）
  - 諾里斯（`4A24 = FF`、槽 0 `4AA6`）
  - 訓練、買甲
  - 索寇要塞（`4AA7 = FF`）
  - 東航線 → 波多廣場北緣 → 斯托亞諾夫城門的馬車 → 城堡內部 GEO5/3→6→5→4→3 → 上樓 block 5 → 7
  - 覲見廳（`4ABA = FE`）

**原版那一側（dosgolem，`workplace/dosgolem`，分支 `pool-parity`，`b4d5a3f`）**：
- `cmd/shots` 目前能 `-peek`／`-trace-peek`／`-trace-call`／`-watch`／`-load-state`／`-save-state`，**還不能寫記憶體**。
- 既有的原版駕駛：
  - `tools/dosgolem-drive-orc-home.py`：逐步讀位置、分類畫面、從狀態檔接著走；
  - `tools/dosgolem-block-load-clear-cases.py`：建隊、走到市政廳、存讀檔。
  - 建隊鍵序可以沿用 `docs/audit/dosgolem-deployment-peek-goblins.json`。
- 隊伍在原版記憶體裡的位置：
  - 隊伍鏈從 `DS:5CF4h`（far pointer）開始，`+104h` 是下一個（spec 040／052／084／088）；
  - 目前 HP 是 `+11Bh`、上限是 `+32h`（spec 049）；
  - 陣營在 `+10Eh`（`tactical.go` 的註解，待核對）；
  - 倒地／死亡的狀態欄位**還沒定位**（spec 018 只有存檔那一側的 `status`）。
  - 戰鬥中怪物也串在同一條鏈上（spec 052），所以鎖 HP 必須只寫隊員。
- ECL 旗標在原版記憶體的換算：class 0 是 `[4933h] + 6E00h + addr × 2`，class 1 是 `[4937h] + 2A00h + addr × 2`
  （spec 106）。建隊之後量過 `[4933h] = 2EA2:0000`、`[4937h] = 2F22:0000`；讀檔或換 overlay 之後要重讀，不能寫死。

## 提示詞（可直接貼給 `/goal`）

> 目標：同一條路線 (a) 在兩邊都作弊跑到結局——remake 開產品的 F6 作弊選單，原版用 dosgolem 鎖 HP——
> 每段留下必經區塊與旗標，兩邊逐段對照；對不上的地方先開 issue 帶證據。做完關 #5。
>
> **A. remake 那一側**
>
> 1. 新增 `TestMainlineProbeCheatMenuToEnding`：
>    - house rule、seed 142、路線 (a)；
>    - 導覽結束、第一次自由移動時用 `setCheats(true, true)` 打開兩個開關，全程不關；
>    - 其餘照 `runMainlineProbe` 的正常按鍵路線，不掛 `hp_lock_test.go` 的治具。
> 2. 探針每跨一段寫一筆 checkpoint：段名、ECL archive／block、地圖、座標、遊戲時刻、隊伍等級、下列旗標：
>    - `4ABB`、`4AC1`；
>    - 槽 `4AA6..4ABF` 裡值為 `FE`／`FF` 的那幾個；
>    - `4A24`、`4AA7`、`4AC4`、`4A77`、`4A78`、`4ABA`；
>    - 這一段換過的區塊序列。
>
>    寫成 `docs/audit/remake-cheat-playthrough.json`（由測試產生，commit 進版控），並記下作弊寫回次數與 `CheatsUsed`。
> 3. 跑到 `4ABA = FE` 與結局頁就算通過；中途出現非戰力的卡點，照舊修治具或開 issue。
> 4. `TestMainlineProbeHPLockedRouteA`（治具版）：
>    - 如果新測試涵蓋了它的所有斷言，就刪掉治具版與 `probeRouteA` 以外不再用到的治具碼，在 spec 137 與 playtest 註明由誰取代；
>    - 不刪就寫明理由。
>
> **B. 原版那一側（dosgolem）**
>
> 5. **在扣血公式的入口攔截**（使用者 2026-09-17 補充：原版有反組譯紀錄，知道計算公式的位置，可以攔截）。
>    - 原版所有扣血都經過 overlay-25 entry 28（`2266h`，spec 084〈狀態轉移〉）。它是 `f(目標: far pointer; 傷害: byte)`：
>      比較 `+11Bh`，把倒地／瀕死／死亡（4／5／6）寫進 `+10Ch`。
>    - 呼叫端（掃 `9A AC 00 0A 01`，正對照是 overlay-13 `16D2h` 的 `0100:003E`，與 spec 050 對得上）有三個：
>      - overlay-03 `2A71h`（ECL `DAMAGE`）；
>      - overlay-13 `048Dh`（戰鬥攻擊）；
>      - overlay-24 `14FBh`（法術）。
>    - 三者都呼叫 resident stub `010Ah:00ACh`。在 stub 上攔，就不管 overlay 被載到哪個段；
>      這時堆疊上 `[sp+4]` 是傷害、`[sp+6]`／`[sp+8]` 是目標的 offset／segment。
>    - IDA 匯出放 `workplace/ida-g5-ov25/`、`workplace/ida-g5-ov13/`；結論寫進 spec 084，附位元組與證據等級。
> 6. **dosgolem 加 `-intercept-damage lock,kill`。**
>    - 在 `cmd/shots` 每道指令執行前（與 `-trace-call` 同一個掛點）檢查 `CS:IP == 010Ah+LoadSeg:00ACh`：
>      - 目標是隊員且開了 `lock`：把堆疊上的傷害改成 0；
>      - 目標是怪物且開了 `kill`：傷害大於 0 時改成目標的 `+11Bh`。
>    - 隊員／怪物的判準：不在戰鬥中（`DS:4954h != 5`）一律是隊員；戰鬥中看 `+10Eh` 的陣營。陣營值由收據量出來，不猜。
>    - 每一次攔截記一筆：步數、陣營、目標 HP、原傷害、改後傷害，寫到 stdout 與 `shots.json`。
>    - 正反對照：從 `workplace/orc-drive/` 衛兵攔截的戰鬥狀態檔開快速戰鬥。不攔要有隊員受傷或倒地；
>      攔了隊員的傷害全部是 0，戰鬥以勝利收場。
>    - commit 在 dosgolem 的 `pool-parity` 分支。
> 7. **原版的分段駕駛。** 寫 `tools/dosgolem-cheat-playthrough.py`：
>    - 以狀態檔分段，每一段從上一段的 `.state` 接著跑；
>    - 走路用讀位置、查 GEO 規劃的方式（沿用 `dosgolem-drive-orc-home.py` 的做法，抽成共用函式）；
>    - 選單照 remake 探針同一段的答案；
>    - 每一段結束時 peek 同一組旗標，換算方式照 spec 106，基底每段重讀；
>    - 寫 `docs/audit/dosgolem-cheat-playthrough.json`，格式與 A.2 相同，外加 generator revision、原版 EXE 的 SHA-256、每段鍵序與幀數。
>
>    段落順序與 A 相同，一段一段往前推：
>    - 每段都要先有收據才往下；
>    - 狀態檔放 `workplace/dosgolem-cheat/`，不進版控；
>    - 收據記下每段狀態檔的 SHA-256。
> 8. 原版的亂數與 remake 不同：隨機遭遇的次數與時刻、戰鬥回合數、時刻，**不列入對照**。
>    對照的只有：
>    - 每段的必經區塊序列；
>    - 段末的委任槽與主線旗標；
>    - 結局旗標與結局頁。
>
> **C. 對照與收尾**
>
> 9. 寫 `tools/cheat-playthrough-compare.py`：讀兩份 JSON，逐段列出區塊序列與旗標「一致／不一致／原版還沒跑到」，
>    輸出 `docs/audit/cheat-playthrough-compare.md`。不一致的每一項，先查 spec 與原版位元組，確認是 remake 的偏差就開 issue 帶證據，
>    不在這一輪順手改產品碼（除非修法是一兩行、證據 exact，且有測試）。
> 10. 文件：
>     - spec 137：路線 (a) 改成「兩邊作弊通關的收據」；
>     - playtest 補十五：兩邊各段的表與對照結果；
>     - CONTEXT.md；本 goal 補〈收在哪〉。
> 11. 台帳：
>     - 兩邊都到結局、對照表沒有未開 issue 的不一致時，關 #5；
>     - 原版只跑到一半時 #5 不關，留言寫明跑到哪一段、下一段卡在什麼；
>     - `docs/worklist.json` 的 #5 條目改綁新測試或對照報表；
>     - 依序跑 `pool-worklist -mode render -write WORKLIST.md` → `-mode verify` → `gh issue list --state open`，雙向核對。
>     - 本 repo 可以 commit 與 push；dosgolem 的 commit 推送前先問。
> 12. 停止線：
>     - 原版某一段換了三種不同的駕駛做法仍過不去；
>     - dosgolem 要寫的欄位定位不出來（第 5 步兩次量測都沒有訊號）；
>     - remake 那一側要改 `internal/combat` 或共用 engine 才過得去；
>     - 需要作弊選單以外的注入；
>     - 單一段超過 30 分鐘。
>
>     停下時寫明卡在哪一段、用過哪些做法、下一步的假設。

## 已知風險與待決

- **原版駕駛的工作量**：路線 (a) 很長（貧民窟 25 場、兩次往返古托井、索寇要塞、東航線、城堡四張圖）。原版沒有 remake 的
  規劃器與畫面識別字，要靠讀記憶體判斷位置、選單與戰鬥狀態。這一輪很可能只推到中段——那也是合格的一輪，
  但 #5 不關。
- **原版的選單辨識**：remake 的 `-screen-state` 在原版裡沒有。`dosgolem-drive-orc-home.py` 用畫面分類
  （`classify`），脆弱；能讀記憶體判斷（例如「目前是戰鬥」「選單等輸入」）就優先讀記憶體，要先找出那幾個位址。
- **陣營值**：`+10Eh` 是 `2266h` 戰鬥中拿來減人數的那一邊（`[6772h + record[+10Eh]]`），隊伍那一邊是哪個值要量。
- **攔截的漏網**：`2266h` 以外若還有直接寫 `+11Bh` 或 `+10Ch` 的路徑（例如毒、能量吸取），攔截蓋不到；
  鎖 HP 收據要記下「戰鬥收場時有沒有隊員不是狀態 0」，有就去找那條路徑。
- **兩邊的時刻不同**：市政廳晚上鎖門、斯托亞諾夫城門的馬車商人晚上不出現，兩邊都要睡到早上。原版的休息流程要另外寫。
- **#42**：remake 在導覽結束那一格休息完會重複印一句，走這條路線時留意，不要誤判成卡點。
- **dosgolem 分支**：`pool-parity` 落後 dosgolem main 22 個 commit，這一輪不合併。
