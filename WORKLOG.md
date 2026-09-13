# Pool of Radiance remake 工作歷程

本檔按日期追加已完成工作的短收據；目前真相見 `CONTEXT.md`，未完成工作與驗收條件
見 `WORKLIST.md`，逆向證據見 `docs/spec/`、`docs/re/` 與 `docs/audit/`。

## 2026-09-13：對拍抽樣加入紮營 REST

- 目標：推進 GitHub issue #8，從未抽樣清單加入可由既有路徑決定性抵達的畫面。
- 基準：延長 dosgolem 主鍵序，在 camp 後送 `R`；`64-r` 的 SHA-256 為
  `127e40a366e8446cd71e95d2024db6998d0f348718f65633c9c306201d8b9c75`。
- 發行包：以 Docker-only 流程從當前原始碼乾淨建立本機
  `v.1.1.3-20260913`；同時把圖示產生器移回容器、版本字串改為正式正規式，
  並修掉雙引號命令中的反引號誤執行。
- 驗證：`tools/appimage-dos-parity.sh` 正常玩家按鍵路徑產生 28 項報表；
  `camp-rest` 與 ALTER、SPEED、PICS、ICON、ORDER 兩步、DROP 全部各有一組
  dosgolem 基準與 AppImage 畫面。issue #8 是滾動清單，`camp-quit` 與其他
  場景仍保留，未宣稱完成。

## 2026-09-13：規格測試反向引用清零

- 目標：完成 GitHub issue #13，逐份分類 56 個 `specs_without_tests`，不以
  無關引用壓低數字。
- 處置：既有相關測試補 `spec NNN`；spec 038、089、094 補直接回歸測試；
  spec 133 以檔頭 `測試：` 明列 dosgolem 外部軌跡。索引器新增
  `specs_tested_outside_go`，不把外部驗證混成 Go 測試。
- 驗證：重生索引後 136 份規格的 `specs_without_tests` 為 0；
  `cmd/pool-doc-index`、`cmd/pool-ecl-memory-audit`、`internal/gamepack` 與
  `cmd/pool-game` 的相關測試全綠。

## 2026-09-13：力量效果收尾與 138 格覆蓋盤點

- 目標：完成 GitHub issue #12 的「先盤點、再接第一個玩家可見代碼」。
- 證據：IDA Pro 9.4 重讀 overlay-12 `04CBh..05E0h` 與 overlay-24
  `10DEh..127Dh`，核實 `0Ch`／`26h` 的 active／inactive 快照與重疊切換。
- 實作：變大術／力量術族掛上有時間的可收尾節點；地圖與戰場到期共同還原
  力量、18 百分位與角色庫，剩餘的較強效果會接管。
- 驗證：純規則測試涵蓋重疊與較弱 inactive 效果；整合測試從 `Update()` 送
  `C`、選法術與目標，確認掛上後十回合到期並由力量 18 回到 10。

## 2026-09-13：探索治具打開門後覆蓋增加

- 目標：完成 GitHub issue #9；門選單不再一律 EXIT。
- 實作：每道門記住已試動作，透過 `Update()` 依序送
  BASH／PICK／KNOCK，全部失敗才 EXIT；治具加入賊與 Knock。
- 驗證：選項順序單元測試通過；同一條世界委任測試由 21 張地圖、19 個 ECL
  block、五條委任、80.29 秒，變為 22 張、19 個、五條、55.13 秒。
  收據與環境記錄在 `docs/audit/explorer-door-coverage.json`。

## 2026-09-13：遊戲內攻略加入第二張地圖

- 目標：完成 GitHub issue #7，把 `F3` 攻略從文明區擴到至少一張其他地圖。
- 證據：`GEO2/20` 的地圖身分與入口由 spec 100／136 閉合；入口後 `(14,4)`
  有 dosgolem 正常路徑收據，五個逃跑落點直接讀自 ECL2/20
  `B6F2h`／`B6F7h`，並以 `cmd/pool-world-graph` 的原始 `cell_terrain`／
  `cell_components` 核對都落在可走區。
- 實作：繁中與英文攻略各新增 `2/20` 貧民窟及六個帶 `source` 的點，更新
  `docs/guide/README.md`；沒有採用第三方攻略座標。
- 驗證：`internal/guide` 的雙語座標、原始 GEO 與重複格三項測試全綠；
  authoritative worklist verify 不再列出 `guide-other-maps`。

## 2026-09-13：怪物特殊攻擊接通能量吸取與恢復

- 目標：完成 GitHub issue #3；定位能量吸取的怪物資料、命中後派發點與
  `55h`／`56h` 處理常式，接到既有欠帳與 Restoration 路徑。
- 證據：`MON2SPC.DAX`／`MON4SPC.DAX` 的 SPECTRE、WIGHT、WRAITH、VAMPIRE
  節點分別帶 `56h`／`55h`；overlay-13 `1740h..174Dh` 以攻擊形態＋1 派發
  群組 2／3；IDA Pro 9.4 證實 overlay-12 entry 80／81 分別傳 1／2 給
  `211Eh`。輸入雜湊與 bytes 收在 `docs/audit/dos-energy-drain-special-attacks.json`。
- 實作：新增 MONnSPC block 解析與怪物效果 staging；命中後依 `55h`／`56h`
  扣一級／兩級，同步職業等級、經驗值、三份 HP 與欠帳；Restoration 同時
  還回等級、HP 與門檻經驗值。
- 驗證：四個原版怪物節點、純規則吸取／恢復往返，以及真正戰術命中路徑
  三層測試通過；`go test ./internal/gamepack ./cmd/pool-game -count=1` 亦通過
  （1.593 秒／298.555 秒）。回歸期間並補齊 #15 後遺漏的測試 `eclSeed=1`：
  固定骰子的路徑現在也固定 ECL RANDOM，世界巡迴不再隨牆鐘漂移。

## 2026-09-13：盜賊訓練技能重算接上原版

- 目標：完成 GitHub issue #2；查清訓練是否再算 `+77h..+7Eh`，以及是否使用
  真實 DEX，補齊 spec 095 與玩家路徑實作。
- 靜態證據：IDA Pro 9.4 釘住 overlay-16 `2F36h` → overlay-23 entry 1；entry 1
  在 `018Dh` 查 `+9Ch` 盜賊等級，`01A4h` 呼叫 entry 4（`031Eh`）。因此訓練
  使用同一張 DEX 表，而且此時讀的是完整角色記錄的真實 DEX。
- 動態證據：dosgolem commit `d351681`，原版 `start.exe` SHA-256
  `12811cbc…`。正常建立的 Dwarf thief 只注入 DEX 18、Gold 5000、XP 1251，
  經正常新遊戲、Rolf 導覽、走路、`ROGUES` 門、TRAIN 與 A 槽存檔後，等級
  `1 → 2`、技能 `23 2D 28 0F 0A 0A 4B 00 → 2D 36 2D 1F 19 0A 4C 00`。
  完整按鍵序、雜湊與位址見 `docs/audit/dos-thief-training-skills.json`。
- 實作：建角仍固定傳 DEX 0；訓練成功後改以角色真實 DEX 重算八格並同步
  角色庫。新增從 `Update()` 送 1、T 的 DEX 18 樣本測試。
- 驗證：兩條訓練焦點測試通過。完整 `cmd/pool-game` 跑到 257.404 秒時只有
  `TestPayingTheTollAtStojanowGateOpensTheCastle` 失敗；單獨重跑仍在馬車旗標
  `@4A77` 失敗，與本次只涉及訓練／技能的資料流無交集，未冒稱整包全綠。

## 2026-09-13：正常新遊戲與固定 ECL seed 分流

- 目標：完成 GitHub issue #15；正常遊玩不再每局走同一條 ECL RANDOM 流，
  測試與對拍仍可明確固定。
- 變更：兩組 ECL session constructor 新增 `WithSeed` 入口；舊 API 與掃描器
  保留 seed 1。`newApp` 以同一個 `UnixNano` seed 建立角色擲骰與 ECL 流，
  `-dice-seed` 同時覆寫兩者。讀檔仍保留 snapshot 的隨機消耗位置。
- 驗證：synthetic constructor、兩 seed 同路線遭遇、正常新遊戲 seed 傳遞
  三條目標測試通過；完整 `internal/gamepack` 1.910s，完整
  `cmd/pool-game` 277.391s，皆以專案 Docker／Xvfb 工具鏈通過。

## 2026-09-13：測試按鍵的同 tick 語意對齊 Ebitengine

- 目標：完成 GitHub issue #14，讓 `scriptedKeys.JustPressed` 不再藉由查詢時
  `delete` 假裝按鍵消費。
- 勘誤：先只改成不消費，兩條 `TestNormalKeys*` 皆在建角結束後得到
  0 人隊伍。原因不是產品分派，而是共用的 `scriptedTextKeys` map 把每個
  舊 tick 的鍵都留著。
- 變更：`JustPressed` 在同 tick 內不消費；兩條跨 tick 流程的 `step`
  每次建新鍵集合，`idle` 清成空集合。產品程式碼無需修改。
- 驗證：兩條正常按鍵測試單獨通過（0.190s）；完整 `cmd/pool-game`
  回歸通過（353.065s）。兩者都使用專案既有 `coab-go-test:20260729`
  Docker／Xvfb 工具鏈。

## 2026-09-03：離場者的先攻、以及歷史信箱改寫

- 目標：查 `NEWECL FF` 修完、GEO1/31 走得進去之後冒出的兩類卡住，先分清是既有
  問題還是敵方 AI 改版新引入。
- 變更：`startRound` 不再給離場者（體型 0）先攻分數，骰子照擲、結果丟掉；
  `RequiredFacing` 對盤面外座標當場失敗，並訂正「DirectionAny 恆真」那句註解。
  巡覽治具的硬失敗訊息補上 seed、狀態列、整份戰術名冊與 ECL block。
  spec 062 加契約 7（離場者不參與先攻）與「runtime `+3` 維護鏈未閉合」一節，
  spec 058 加契約 6（體型查不到時出界檢查整段不生效），spec 107 第八節改寫成現況。
- 驗證：`TestStartRoundKeepsTheFallenOutOfInitiative` 帶正對照，暫時收起修正時紅
  （「打倒的槽在第 1 回合又拿到先攻分數 3」）、修正後綠；世界巡迴 22 趟的硬失敗
  由 3 筆歸零；整包測試 26 個 package 全綠。根因歸屬用 `git log -S` 釘住：
  `state.Scores[index] = score` 只由 `e6fe452`（09-02 的回合迴圈）引入，之後沒人
  改過，三個敵方 AI commit 都沒碰 `startRound`／`selectActor`——**既有問題**。
- 找根因的關鍵不在讀碼，在讓錯誤說得出話：`Update` 把 `tacticalInput` 的錯誤收進
  `a.statusLine`，而治具印的是 `tactical.Status`，於是每個影格重試同一個失敗、
  日誌裡一則錯誤都沒有。治具的失敗訊息要帶上產品碼吞掉錯誤的那個欄位。
- 未完成：GEO7/23 (1,1) 的格子選單卡住。修正後的世界巡迴不再報它，只是那一輪的
  隨機路徑不再經過該區——掃 102 趟才找回一個重現（seed 106、destination 2）。
  形狀與兩個要分開查的疑點記在 `WORKLIST.md`。
- repo 狀態：歷史裡 59 個早期 commit 的 author／committer 是公司信箱，經使用者同意
  以 `git filter-branch --env-filter` 全部改寫成 `wicanr2@gmail.com`（311 個 commit
  全部一致），改寫後重跑整包測試再 force push。repo 無 tag、無 release，不需重建。
  舊歷史留在本機 `backup-before-email-rewrite` 與 `refs/original/`（**不得推送**）。
  文件裡引用 Pool 自己的兩個 commit hash 已對回新值；其餘引用屬共用 engine 或是
  證據檔的 SHA-256 前綴，不受影響。
- 歷史裡八個爆掉的 `WORKLIST.md` blob（13～88 MB，原始合計 458 MB／磁碟 4.8 MB）
  也清掉了：`filter-branch --index-filter` 把大於 1 MB 的 `WORKLIST.md` 換成壞掉前
  最後的正常版本（29 KB），語意等於「那八個 commit 沒更新 WORKLIST」——與當初
  `aa6b934` 的還原做法一致（它是從 `215f8fa` 取回清單，再把期間真正新增的項目併回）。
  成因是 AGENTS.md §9 記的空錨點 `str.replace`。**內容沒有遺失**：抽驗確認壞檔裡的
  插入物（`29h ENCOUNTER MENU`）就在還原版第 296 行。
  驗收：逐一配對改寫前後的 313 個 commit，`git diff --name-only` 只出現
  `WORKLIST.md`，其餘檔案一個都沒動；HEAD 的工作樹內容完全相同；整包測試綠；
  push 不再出現 GitHub 的大檔警告。舊歷史留在 `backup-before-blob-cleanup`。
  空間要等刪掉兩個 backup branch 與 `refs/original/` 再 `git gc` 才會回收。
- Docker 清理：所有 Go／Xvfb 容器皆由 `tools/go.sh` 的 `--rm` 結束，無殘留。

## 2026-09-02：README 現況與截圖收據更新

- 目標：讓專案入口反映第一場 Slums 遭遇、Spec 048～053 與剩餘工作，重新驗證 README
  操作畫面，並登記下一輪 Journal 全文翻譯。
- 變更：修正 DOS ZIP 檔案數與 ECL VM 過期斷言；加入有明確分母的 20～25% 保守估計、
  Slums 戰鬥前 staging 能力與限制；`WORKLIST.md` 新增《軟體世界》說明書 Journal 翻譯
  的來源、術語及零遺漏驗收條件。
- 截圖：Docker／Xvfb 從真實 Ebitengine 視窗逐鍵重拍姓名輸入到 Rolf 導覽七張圖；輸出
  與版控 PNG 逐 byte 相同。新增 `docs/audit/remake-screenshot-manifest.json` 保存來源提交、
  日期、960×600 尺寸、SHA-256 與「正常 remake 玩家路徑、非 DOS parity」標籤。
- 驗證：七張 PNG 雜湊／尺寸／唯一性與 README 本機連結全部通過；`git diff --check`
  通過。嘗試由單元測試離屏 image 輸出戰鬥 staging 時，Ebitengine 因正式 command queue
  尚未建立而拒絕；已完整撤回該測試變更與暫存輸出，未加入 production direct-entry。
- 未完成：正常玩家路徑尚未抵達可操作戰術畫面，因此沒有戰鬥 runtime 截圖；下一輪先
  完成 Journal 翻譯 corpus，再進行 UI 接線。
- 提交／推送：本輪文件、manifest 與工作歷程提交至 `main`，並推送 `origin/main`。
- Docker 清理：所有一次性 Xvfb／Go／檢查容器皆以 `--rm` 結束，無專案容器殘留。

## 2026-09-13：#8 的 35 項發行包對拍

- 目標：完成不需要真人判讀的對拍缺口：紮營九層、商店、神殿、挑法術、
  裝備、地圖施法與平面圖。
- 原版：以 dosgolem 正常建角／導覽鍵序重生 camp-quit、神殿、商店與人類
  牧師法術頁的隔離基準。所有來源都寫 `provenance.json`；比較腳本逐一驗
  generator、dosgolem revision 與同一份 `start.exe` 雜湊。
- remake：從本機未發布的 `v.1.1.4-20260913` full-local AppImage 跑兩個
  正常玩家流程。第一局走地圖進設施；第二局建立人類牧師、記憶祝福術、
  紮營兩小時，再由地圖 C 鍵施法。沒有 direct-entry、座標注入或 ready
  法術注入。
- 收據：`docs/audit/dos-parity-sample.json` 共 35 項；標題 99.79%，
  第一人稱視野 100%，新增法術頁／裝備／地圖施法法術頁外框分別為
  94.23%／96.57%／99.27%。神殿、商店、裝備與施法挑人均明標
  `layout-only`，不把不同資產或 remake 額外畫面冒稱 exact-state。
- 台帳：先從 `docs/worklist.json` 移除 #8，再由
  `cmd/pool-worklist -mode render -write WORKLIST.md` 重生 `WORKLIST.md`。結局留給人工
  issue #5；原版沒有的 `view-pick`／`menu-drop-confirm` 不列為漏抽。

## 2026-09-13：GitHub issues 改為工作主台帳

- 目標：依使用者定案，將未完成工作、狀態與討論的權威由本地 JSON 改為遠端
  GitHub issues；`docs/worklist.json` 只保留 `verify` 輔助鏡像職責。
- 變更：更新 `AGENTS.md`／`CLAUDE.md`、`CONTEXT.md`、`WORKLIST.md` 與 JSON 說明；
  `pool-worklist` 現在拒絕缺少、無效或重複的 `github_issue`，避免本地新增未登記工作。
- 盤點：主機 `gh auth status` 有效；遠端 open issues 為 #1、#4、#5、#6、#10、#11。
  `WORKLIST.md` 全檔只有同六個未勾選項，JSON 也只有同六筆，沒有孤兒工作或映射缺口。
- 驗證：`tools/go.sh test ./cmd/pool-worklist` 通過；`gofmt -d` 無輸出；
  `-mode verify` 只列上述六項；舊「JSON 是權威」現行文字零命中，`git diff --check`
  通過。一次性容器均以 `--rm` 結束，專案樹無 root-owned 檔案或誤建的 `.md` 目錄。

## 2026-09-14：#10 城堡覆蓋測試效能

- 基準：同一 Docker 包裝器下單跑 `TestTheCastleBehindStojanowGateHasContent` 為
  199.52 秒，完整保留 ECL block 3、4、5、6、7。
- 根因：正常付費過門只需 0.25 秒；rotate 3 命中 block 5／7 並離開城堡後仍繼續
  掃城區，分段耗時 166.69 秒。rotate 1／2 沒有增加要求覆蓋。
- 變更：只保留會增加覆蓋的 rotate 0／3，命中各自目標後立即停止；南緣 block 6
  仍走正常地圖生命週期。新增每次不得超過 120 秒的測試內閘門。
- 驗證：同一容器連跑三次為 2.08、1.92、2.59 秒，三次都取得 block 3..7；
  相鄰三條正常路徑測試 18.863 秒通過，`./cmd/pool-game` 全套 204.451 秒通過；
  可重生收據在 `docs/audit/castle-test-runtime.json`。
- GitHub：修正 `8350f28` 已推送；證據回覆後關閉 issue #10，才同步移除
  `docs/worklist.json` 的輔助條目並重生 `WORKLIST.md`。
