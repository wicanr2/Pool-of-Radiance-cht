# Goal：載入 ECL 區塊時清暫存，接著把 HP 鎖定探針從交件跑下去（GitHub #41 → #40；#26、#22 → #5）

主台帳：[issue #41](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/41)（載入區塊時沒清 `4A00..4A1F`／`6E79..6E82`）。
接續：[#40](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/40)（HP 鎖定診斷通關，被 #41 擋住）。
相關：[#26](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/26)（路線 (a)）、
[#22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（正常強度的策略層）、
[#5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)（正常強度通關，維持 open）。
上一個 goal 的結案位置：[`issue-40-hp-locked-route-a.md`](issue-40-hp-locked-route-a.md)〈2026-09-17 收在哪〉
（停在停止線：要改產品程式才過得去）。

## 為什麼先做這一條

HP 鎖定探針（`TestMainlineProbeHPLockedRouteA`）已經走通這幾段：貧民窟 25 場、交件、職員列出波多廣場、
拍賣結案、打贏諾里斯。接下來兩道**不是戰力造成的**閘門過不去，原因都是 `4A01` 帶著舊值：

- 職員 `ecl3/8 9BA1h`：`4A01 > 0` 時走 BACK SO SOON，不掃獎賞槽，波多廣場與諾里斯的件都交不出去。
- 港務長 `ecl3/0 9C4Bh`：`4A01 == 1` 時直接 EXIT，拿不到去索寇要塞的票。

原版不會卡在這裡。每載入一個 ECL 區塊，引擎就把那兩段位址清成 0；remake 沒有做這一步。
這是 remake 的缺陷，不是駕駛的問題，也不是戰力的問題。

## 現況（2026-09-17，`c70b611` 已推送）

**原版行為**（spec 106〈載入區塊時清掉的兩段〉）：
- overlay-07 entry 3（`01C8h..0333h`）在 `02D8h` 檢查 `[4959h] == 0`，成立時做兩件事，位元組 exact：
  - class 0 的 `4A00..4A1F` 清成 0（迴圈 `add ax, 49FFh`）；
  - class 1 的 `6E79..6E82` 清成 0（迴圈 `add ax, 6E79h`）。
- `[4959h]` 非 0 時跳過清除，並把它歸零。寫 1 的是 overlay-16 `03EFh`，推測是讀檔（strong inference）。
- 同一支在前面還有別的寫入，#41 內文有列：`6DE1 = FFh`（位移 `+5C2h`）、`6DD2`／`6DD3`、`49E5` 清 0、`49E6 = 1`。
- dosgolem 收據 `docs/audit/dosgolem-4a01-block-load-clear.json`：走出市政廳時 `4A01 01→00 ← 1997:02FE`。
  只量了這一種換區。產生工具是 `tools/dosgolem-4a01-block-load.py`，dosgolem 版本 `pool-parity` 分支的 `b4d5a3f`。

**remake 的換區**：
- 發生在共用 engine 的 `eclvm.BlockSession.switchTo`（`RunUntilEvent` 內部），流程是：
  `SwitchBlock` → 新區塊入口 0 → `pendingEntries`。
- 作品端只有 `SetBlockCatalogResolver` 這一個掛點。它拿到的是**記憶體複本**，
  而 engine 的註解明文規定 callback 不得改 VM 狀態。
- 清除要發生在 `SwitchBlock` 之後、新區塊的入口開跑之前。等 `RunUntilEvent` 回來再清就晚了：
  新區塊的入口可能已經寫進範圍內的位址。
- 所以照 CLAUDE.md §4「game pack 描述不了就先擴充 engine」，需要一個**作品中立**的載入宣告。

**其他換 session 的路徑**（沿用 CLAUDE.md §9「建／換 machine 的路徑要一起帶」的教訓）：
- `gamepack.NewDOSECLArchiveSession*`（`internal/gamepack/ecl_catalog.go`）
- `intro.go` 的兩個 `eclvm.NewBlockSession`
- `restoreCampaign`（`cmd/pool-game/main.go:2389`，讀檔）
- `configureEventSession`（`main.go` 約 1609 行，接 resolver）

**補洞的 adapter**：`applySokalHandInTicketState`（`main.go:3019`）只在 ECL3/8 把 `4A01` 清回 0。
清除接上之後，這一支應該可以拿掉。

**engine repo 的狀態**：
- `main` 已 fast-forward 到 `7a81a33` 並推送，原本 `postwall-black-only-note` 分支上的 12 個 commit 都在裡面。
- Pool 的 `go.mod` 還鎖 `c207368`；本機建置由 `tools/go.sh` 以 replace 指到本機 engine。
- CoAB 也還鎖 `c207368`（檔案型 proxy `tools/engine-proxy.sh`）。它對新 engine 的抽驗登記在
  [CoAB #1](https://github.com/wicanr2/Curse-of-the-Azure-Bonds-cht/issues/1)。

**全套紅燈六條**（`c70b611`）：
- `TestSokalKeepOpensTheOtherBoatRoutes`
- `TestWorldTourReachesTheAreasBehindTheHarbour`
- `TestPlayingTheWorldCompletesCommissionsOnItsOwn`
- `TestMainlineProbeNaturalPartyFirstBattle`
- `TestMainlineProbeHouseRuleCommissionExperience`（seed 142 在索寇 (8,5) 全滅）
- `TestMainlineProbeHPLockedRouteA`（停在 `route (a) blocked by #41`）

前三條是探索器走碼頭與索寇，log 裡看得到 `4A01=1`。它們會不會因為這次修正而改變，沒有驗證過，不要預設會轉綠。

**本地鏡像**：`ecl-block-load-scratch-clear` 的 verify 綁在探針裡這行自承的 fatal：`route \(a\) blocked by #41`。

## 提示詞（可直接貼給 `/goal`）

> 目標：remake 載入 ECL 區塊時照原版 overlay-07 entry 3 清掉 class 0 `4A00..4A1F` 與 class 1 `6E79..6E82`
> （讀檔後第一次不清），關掉 #41；然後把 HP 鎖定探針（#40）從交件那一段接著往結局跑。
>
> 1. **先補證據，再動程式。** 用 dosgolem 多量兩種換區：
>    - 同一個 archive 內走邊界，例如城區 (0,4) → 貧民窟；
>    - 跨 archive，例如貧民窟 → 古托井，或碼頭上船。
>
>    每一種都要有兩樣東西：
>    - 正對照：換區前 `4A01` 或範圍內另一格非 0；
>    - 換區那一步的寫入監看。
>
>    用 `tools/dosgolem-4a01-block-load.py` 擴充或另寫一支，收據寫進 `docs/audit/`，記 dosgolem revision。
>    讀檔那一次（`4959h`）也試著量：在範圍內有非 0 值時存檔、讀檔，看值是否留住。
>    量不到（兩次嘗試都存讀檔失敗）就記成 strong inference，說明原因，不要卡住。
>    同一支 entry 3 的其他寫入（`6DE1 = FFh`、`6DD2`／`6DD3`、`49E5` 清 0、`49E6 = 1`）逐條盤點：
>    remake 換區後各是什麼值、由誰寫。有差異的先開 issue 帶證據，不要併進這一條，除非證據 exact、
>    而且同一個宣告就能表達。
> 2. **engine：作品中立的區塊載入宣告。**
>    - 在 `golden-box-remake-engine` 的 `main`（`7a81a33`）上開分支，加一個向後相容的 `BlockSession` 設定。
>      名字自己取，語意是：「每次 `NEWECL` 成功換區、`SwitchBlock` 之後、第一個入口跑之前，把宣告的位址設成宣告的值」。
>    - 預設不宣告時行為完全不變。
>    - 用資料表達（位址範圍＋值），不收 callback，保住「callback 不改 VM 狀態」的契約。
>    - `Clone`／snapshot 要把宣告一起帶上。restore 的時候由 adapter 重建，跟 resolver 同一個模式。
>    - engine 程式碼與 example 不得出現 Pool 的位址或名稱。
>    - 合成 fixture 的單元測試要涵蓋：換區後清掉；入口在清除之後寫的值留得住；未宣告時不動；Clone 帶得過去。
>    - engine 自己的測試全綠。
> 3. **CoAB 唯讀回歸。**
>    - 不改 CoAB 任何檔案。先用 `7a81a33` 跑一次 CoAB 現行測試記基準（CoAB #1 的結果能直接用就沿用），再指向新 engine commit
>      （臨時 modfile 或 engine-proxy，產物放 `workplace/`）跑一次。
>    - 兩次的紅燈名單要逐條相同；不同就停下來查，不要改 CoAB。
> 4. **Pool adapter 接上。**
>    - 在所有建立或接上 event session 的路徑宣告兩段清除：新遊戲、intro 交接、`NewDOSECLArchiveSession*`、
>      `configureEventSession`。
>    - 讀檔（`restoreCampaign`）不因為重建 session 而清，這就是 `4959h` 的語意；照第 1 步量到的結果決定，並寫測試釘住。
>    - 位址常數放 `internal/gamepack`，註解帶 spec 106 與原版位址。
>    - 測試從 `Update()` 送按鍵：職員格 → 走出市政廳，斷言 `4A01` 回到 0，並對上第 1 步的 dosgolem 收據；
>      再加一條「在市政廳裡存檔、讀檔，值留住」（或照第 1 步的結果改寫）。
>    - 然後拿掉 `applySokalHandInTicketState`：相關測試與探針都過才拿；拿不掉就寫明是哪一條擋住。
>    - 產品碼改了，`cmd/pool-game` 與 `internal` 的非測試 diff 會有內容。這一輪允許，但只限清除與移除補丁這兩件事。
> 5. **重跑紅燈與探針。**
>    - 全套測試，六條紅燈逐條記：轉綠、原因變了、或原因不變。
>    - `TestMainlineProbeHPLockedRouteA`：拿掉 `route (a) blocked by #41` 那道 fatal 與兩處 `hand-in blocked` 的 note
>      分支，改回真的交件，從波多廣場與諾里斯交件 → 訓練所升級 → 港務長拿票 → 索寇 → 第 6～11 段往結局跑。
>      停止線照 #40 goal 第 9 步。
>    - 沿途每一個非戰力卡點：駕駛問題修治具；remake 與原版不符先開 issue 帶證據。
>    - house rule 與原版規則兩條自然強度探針也重跑，記錄結局有沒有變。
> 6. **對拍。** 這次改的是遊戲狀態，不是繪圖，但玩家看得到的對話會變（港務長、職員）。
>    照 CLAUDE.md §7：`tools/package-release.sh` 建當前原始碼的發行包，再跑 `tools/appimage-dos-parity.sh`，
>    把變好、變差、不變的數字寫進 commit message。對拍腳本拒跑時照它的訊息處理，不要繞過。
> 7. **文件與收尾。**
>    - spec 106：狀態行的 remake 註記改成已接上，寫 `4959h` 量到的結果。
>    - spec 102：「只在 remake 成立」的段落改寫成現況。
>    - spec 137：刪掉死路表標「（remake）」的四列，〈路線 (a) 的 HP 鎖定診斷〉補上新段落的數字。
>    - CONTEXT.md、playtest 補十三、本 goal 補〈收在哪〉。
>    - `tools/go.sh fmt` 只排新檔；既有檔用 `gofmt -d` 看自己改的行。
>    - 暫時的掃描與追蹤碼刪乾淨；用腳本改檔前先驗錨點唯一。
> 8. **台帳。**
>    - 修好並有證據才關 #41，留言附測試名、收據與 commit。
>    - #40 留言交代各段落的旗標、卡點與下一個最小工作；跑到結局才關 #40。
>    - #26、#22、#5 各留一句狀態。
>    - 鏡像：移除 `ecl-block-load-scratch-clear`，更新 #40 條目；新 issue 先建 GitHub 再寫鏡像。
>    - 依序跑 `pool-worklist -mode render -write WORKLIST.md` → `-mode verify` → `gh issue list --state open`，雙向核對。
>    - **engine 與 Pool 的 push、engine 合併進 main、Pool `go.mod` 更新鎖定版本，都要先問使用者。**
> 9. **停止線。** 碰到下列任一情況，停下來回報：
>    - 第 1 步量到原版在某種換區**不清**（跟 spec 106 衝突）→ 先改 spec，再重新設計宣告的範圍。
>    - CoAB 回歸的紅燈名單變了。
>    - engine 的改動要破壞既有 API 或 callback 契約才做得出來。
>    - 同一個非戰力卡點第三次出現。
>    - 鎖定下出現無法結束的戰鬥。
>    - 單一段超過 60 秒，或整條探針超過 10 分鐘。
>    - 需要注入旗標、座標、時鐘、金錢或 XP 才走得下去。

## 2026-09-17 收在哪

第 1～8 步做完，#40 的探針從標題跑到結局。使用者確認後 engine `block-load-writes` fast-forward 併進 main
（`f9c0ae7`）並推送，Pool `go.mod` 鎖這一版（`cebe5ad`，無 replace、`-mod=readonly` 建過），#41、#40 關閉。

- **第 1 步（原版收據）**：`tools/dosgolem-block-load-clear-cases.py` → `docs/audit/dosgolem-block-load-clear-cases.json`（ok）。
  跨 archive（貧民窟 → 城區）class 0 `4A00 FF→00 ← 1997:02FE`、class 1 `6E7D 0B→00 ← 1997:0323`，之後城區入口才寫
  範圍內的值；讀檔那一條路不寫 `4959h`、讀回 `4A01 = 1`，讀檔後第一次換區照常清（`2BA4:02FE`，段內容等於 overlay-07）。
  entry 3 另外五個附帶寫入（`6DE1=FF`、`6DD2`／`6DD3=0`、`49E5=0`、`49E6=1`）位元組 exact，remake 原本一個都沒有做——
  同一個宣告表達得了，併進這一條；`49E6 = 1` 原本只在新遊戲手寫一次。
- **第 2 步（engine）**：分支 `block-load-writes`，`f9c0ae7`：`MemoryFill`、`SetBlockLoadWrites`、`ApplyBlockLoadWrites`；
  `SwitchBlock` 之後、第一個入口之前套用，Clone 帶著，範圍碰到程式碼窗就拒絕。engine 全套測試綠。
- **第 3 步（CoAB 唯讀回歸）**：`git archive` 匯出 `7a81a33` 與 `f9c0ae7`，臨時 modfile 以 replace 指過去，各跑一次 CoAB
  全套測試：兩次都是 60 個套件全綠，紅燈名單相同（零條）；CoAB 工作樹零變更。
- **第 4 步（Pool adapter）**：`internal/gamepack/block_load.go`，三個建構子都宣告；新遊戲前端改呼叫
  `ApplyBlockLoadWrites`。測試 `TestLeavingCityHallClearsTheBlockScratch`、
  `TestLoadingInsideCityHallKeepsTheScratchUntilTheNextBlock`（負對照：拿掉宣告兩條都紅）。`applySokalHandInTicketState`
  與它的單元測試移除。
- **第 5 步（紅燈與探針）**：`TestMainlineProbeHPLockedRouteA` 走到結局（補血 849 次、死亡復活 0）。路上修了兩處治具：
  付不出旅店錢改去貧民窟屋內睡；探索器只在 `4AA7` 變化時重置走近過的地點（`TestDirectedExplorationReachesMaps`
  一度掉到 2 張圖，修完回到 3 張）。全套紅燈五條，名稱同上一輪；兩條自然強度探針的停止位置變了，見 playtest 補十三。
- **第 6 步（對拍）**：`v.1.1.9-20260917` 發行包。對拍腳本在野外施法兩張之前停下：導覽結束那一格休息完又印一次
  導覽結尾句，v.1.1.5 之後就有、與 #41 無關，開 #42。拍到的 33 張用同一支比對程式量，與 v.1.1.5 逐項相同
  （變好 0、變差 0、不變 33）；`docs/audit/dos-parity-sample.*` 沒有覆寫。
- **第 7 步（文件）**：spec 106 狀態 CONFORMED、附帶寫入表與收據；spec 102 刪掉只在 remake 成立的順序限制；
  spec 137 死路表刪四列、鎖血診斷節改成走到結局；CONTEXT、playtest 補十三。
- **第 8 步（台帳）**：開 #42；#41、#40 推送後關閉並從鏡像移除；#26、#22、#5 各留狀態。

## 已知風險與待決

- **每種換區是否都經過 entry 3**：目前只量過「走出市政廳」這一種。碼頭上船走的是 `LOAD FILES` 再 `NEWECL`；
  野外與城鎮的換圖由各區塊自己的入口帶 `6E12`。第 1 步不做完，就不要把清除宣告寫成「所有 NEWECL」。
- **讀檔語意**：`4959h` 的寫入者在建隊選單那一支，「讀檔後第一次載入不清」是推論。remake 的 `restoreCampaign`
  不走 `switchTo`，本來就不會清，所以風險在反方向：讀檔之後的**下一次**換區必須照常清。
- **`6E79..6E82` 是入口常用的暫存**（遭遇選單結果、表驅動戰鬥的列號）。清除的時點必須早於入口；
  否則會抹掉入口剛寫的值，症狀是遭遇選單或戰鬥編成錯位，而且看起來像別的 bug。
- **探索器的三條紅燈**（碼頭、世界巡迴、自動完成委任）可能改變行為，也可能是別的原因。逐條看 log，不要把轉綠
  或轉紅都算到 #41 頭上。
- **自然強度的牆沒變**：諾里斯約 15% 勝率，索寇 (8,5) 19 個 seed 全滅。這一輪就算鎖血跑到結局，也只是 #40 的
  診斷收據，#5 維持 open。
- **engine 分支**：新 API 從 `main` 開分支做，併回 `main` 要使用者決定。Pool 與 CoAB 的 `go.mod` 都還鎖 `c207368`：
  CoAB 升版走 CoAB #1；Pool 升版時要一起確認 `7a81a33` 與新 API 兩段。
