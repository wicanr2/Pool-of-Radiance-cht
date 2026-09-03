# Pool of Radiance remake 工作歷程

本檔按日期追加已完成工作的短收據；目前真相見 `CONTEXT.md`，未完成工作與驗收條件
見 `WORKLIST.md`，逆向證據見 `docs/spec/`、`docs/re/` 與 `docs/audit/`。

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
