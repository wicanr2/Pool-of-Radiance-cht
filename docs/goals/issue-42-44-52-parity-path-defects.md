# Goal：把對拍那條路上的三個缺口補掉——導覽格休息重印（#42）、索寇回程船落點（#44）、戰利品截圖（#52）

主台帳：[issue #42](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/42)（導覽結束那一格休息完又印一次結尾句，
發行包對拍停在野外施法前）、[issue #44](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/44)（索寇要塞回程船
落在貧民窟，原版是城區碼頭）、[issue #52](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/52)（發行包拍
remake 的戰利品畫面）。
上一個 goal 的結案位置：[`issue-49-51-52-current-state-calibration.md`](issue-49-51-52-current-state-calibration.md)〈2026-09-18 收在哪〉。

## 為什麼這三條放同一輪

三條都卡在「**對拍與截圖走不完整條路**」這一件事上：

- #42 讓對拍每次都停在第 33 張，`field-cast` 兩張自 v.1.1.5 之後再也沒有量過——而它同時是一個
  **玩家看得到的缺陷**（休息完畫面停在導覽的結尾句）。
- #52 缺的是 remake 側的戰利品截圖，上一輪已經把路線縮到「貧民窟腳本的 7 處 `27h TREASURE`」。
- #44 是兩邊逐段對照唯一還開著的 remake 行為偏差，證據齊全（原版位元組 exact），修完那張對照表就乾淨了。

上一輪剛把量尺修好（`cmd/pool-parity-check` 對的是基準表，不是上一次跑的結果）。這一輪把量測面補滿，
表上的 35 項才是真的 35 項。

## 現況

**#42（bug）**

新遊戲導覽結束後站在城區 (0,4)，紮營休息 2 小時（`e`、`r`、`h`、`i`、`i`、`r`）：休息結束沒有回到冒險畫面，
停在格子文字，內容是導覽的結尾句。v.1.1.5 走得完、v.1.1.9 起走不完，所以是那之間的行為改變；
已排除 #41（拿掉區塊載入寫入的宣告，結果相同）。**原版在同一格休息完會怎樣還沒量**——
`ref-spells` 基準走的是 `e,m,m`，沒有休息。

**#44（remake 行為偏差，證據 exact）**

`ecl4/21` 入口 0 的 `9977h` 一段寫 `C04B=15`／`C04C=1`／`C04D=3`，再 `SAVE 3 @6E12`、`NEWECL 0`
——`6E12=3` 指定封存檔 3，落點是 ECL3/0 碼頭 (15,1) 朝西。remake 目前落在 ECL2/20 (15,1)。

**#52（截圖）**

`tools/capture-treasure.sh` 已能建六人隊、走到遭遇、打完（`n` 結束戰鬥；`y` 會無限重開空回合）。
沒拍到的原因：隨機遭遇不呼叫 `27h TREASURE`。`cmd/pool-ecl-trace -archive 2 -block 20` 掃出 7 處
TREASURE：`A390h`、`A4ACh`、`A75Ah`、`AABCh`、`AB2Fh`、`AD03h`（另一處在後面）。

## 提示詞（可直接貼給 `/goal`）

> 目標：修掉 #42 與 #44，拍到 #52 的四張，讓發行包對拍走完 35 張。做完關這三條。
>
> **A. #44 索寇回程船（先做，它最小而且證據齊全）**
>
> 1. 從 `ecl4/21` 入口 0 的 `9977h..998Fh` 確認 remake 這一段怎麼走：`C04B..C04D` 有沒有同步、
>    `6E12` 有沒有被讀進「下一個封存檔」。**先讀 spec 再改**（`grep -rl 'spec 0' 找 NEWECL 那一份）。
> 2. 修到「答 YES 之後 `eclArchive == 3`、區塊 0、GEO3/0 (15,1)、朝西」。
> 3. 測試從 `Update()` 送鍵（不要直接呼叫換圖函式）；重跑 `TestMainlineProbeCheatMenuToEnding` 與
>    `tools/cheat-playthrough-compare.py`，`sokal` 段的「出口不同」要消失。
>
> **B. #42 導覽格休息完重印結尾句**
>
> 4. **先量原版**：用 dosgolem 在導覽結束那一格（城區 (0,4)）休息 2 小時，收據寫進
>    `docs/audit/` 並記下鍵序與狀態檔 SHA-256。原版怎麼收尾是這一條的裁判。
> 5. 最小重現：從 `Update()` 送同一串鍵，斷言休息結束後的畫面識別字。**不要用整合測試追**——
>    上一輪的教訓是症狀隨參數改變方向時就該寫最小重現（CLAUDE.md §9）。
> 6. 修：查休息收尾是不是重跑了這一格的入口，或把上一則文字框重新顯示。改完 remake 要與 A.4 量到的
>    原版一致，並留一條測試釘住。
> 7. 打包後跑對拍，確認**走完 35 張**；`field-cast` 兩張回到表上（用 `cmd/pool-parity-check -write`
>    更新基準表，並在 commit message 寫出那兩張的數字）。
>
> **C. #52 戰利品截圖**
>
> 8. 把 ECL2/20 那 7 處 `27h TREASURE` 反查成座標（`cmd/pool-world-cell-sweep`／`pool-initial-cell-sweep`
>    那一類，或直接看入口 1 的分支條件），挑最靠近起點的一處。
> 9. `tools/capture-treasure.sh` 改成走到那一格（穿牆已經開著，路線不必繞），拍四張：頂層、
>    `Take: Money`、`View` 人物頁、分完之後的頂層。
> 10. 與原版六張並列放 `docs/screenshots/treasure/`，差在哪寫進 spec 034。
>
> **D. 台帳與收尾**
>
> 11. #42、#44、#52 以證據關閉；`docs/worklist.json` 三條移除，重生 `WORKLIST.md`，再跑 `-mode verify`
>     與 `gh issue list --state open` 雙向核對。
> 12. 動到玩家看得到的畫面就打包後對拍（CLAUDE.md §7），數字寫進 commit message；
>     這一輪的對拍會自己對基準表，有差就會非零退出。
>
> **E. 停止線**
>
> - #42 的原版量不出來（dosgolem 在那一格休息不了）→ 記下做過什麼，改以「remake 內部一致」為驗收，
>   原版那一側另開 issue。
> - #44 改完會動到 `NEWECL` 的共用路徑 → 停下，先確認 CoAB 不受影響（CLAUDE.md §4 的零干擾）。
> - #52 再換一種做法仍拍不到 → 留腳本與失敗原因，issue 繼續開著。
> - 單一步驟超過 30 分鐘。

## 已知風險與待決

- **#42 可能是「休息推進時間 → 重跑入口」的正常行為**，原版也這樣。所以第 4 步的原版量測要先做；
  先改 remake 會把「與原版一致」改成「與原版不一致」。
- **#44 動到的是換封存檔那條路**，Pool 與 CoAB 共用 engine 的 `NEWECL` 語意；只改 Pool 的 title adapter，
  不要動 engine 的判斷。
- **對拍現在會非零退出**：修完 #42 之後第一次跑會列出 `field-cast` 兩張「表上沒有、這次有」，那是預期的，
  用 `-write` 收進表。
- **#50（考古圖鑑）仍不在這一輪**：那是內容產出，另外排。

## 2026-09-18 收在哪

- **A（#44，關閉）**：旗標清除時機。兩條試過退掉的路寫進 commit 與 spec 100：區塊載入時清（拆掉城堡那一串）、
  換區塊之後清（太晚，新區塊的入口在同一次 `RunUntilEvent` 裡就讀了）。
- **B（#42，關閉）**：先用 dosgolem 量原版（`tools/dosgolem-tour-cell-rest.py` →
  `docs/audit/dos-tour-cell-rest.json`），量到的是城衛隊打斷；remake 的缺口在 `3Ah DELAY` 不算呈現邊界。
  對拍從 33 張走到 **34 張**，`field-cast` 回到表上；`field-cast-spell` 拍不到是原版規則
  （城區街上記不成法術），記成未量。
- **C（#52，續開）**：這一輪驗掉了上一輪的線索（7 處 TREASURE 不是走到格子就有），
  並在 issue 裡留下三條依成本排序的下一步。
- **停止線**：#52 照停止線收手（第二種做法仍拍不到）。其餘沒有碰到。
