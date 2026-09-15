# Goal：主線從開場到結局連續跑過一次（GitHub #5）

主台帳：[issue #5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)。
本文件是給下一個 agent session 的開工提示詞；進度、證據與關閉狀態一律回到 issue，
這裡不重複記流水帳。

## 現況

以 GitHub #5 的最新留言與 [`docs/playtest/mainline-end-to-end.md`](../playtest/mainline-end-to-end.md)
為準；路線本身在 [spec 137](../spec/137-mainline-route.md)。2026-09-15 第三輪之後：
路線層從標題到結局的狀態機已經接通（commit `c713d2c`），但戰後生命值寫回之後，
正常建角的一級隊伍在貧民窟第 3～4 場全滅——下一段是測試駕駛的玩家策略層（#22）。

## 提示詞（可直接貼給 agent）

> 以 GitHub issue #5 為主台帳，讓《Pool of Radiance》remake 從標題畫面用正常玩家按鍵
> 一路走到結局過場，並把每一個必經 block 的經過紀錄留成可重跑的測試收據。
>
> **第 0 步：先收工作樹。** 逐檔審閱目前未提交的 8 個修改檔與 3 個新檔，確認每一段都
> 屬於 #5（不屬於的另開 issue 或還原），跑 `tools/go.sh gofmt -d <檔案>` 只修自己的行，
> 然後提交。commit message 要寫明：這條主線測試目前在哪一段失敗、失敗形狀是什麼
> （ECL 18／港務長循環、5 分鐘上限）；**不得把「快通了」之類的猜測寫成結論**。
> 提交前確認 `git config user.email` 是 `wicanr2@gmail.com`。
>
> **第 1 步：把主線路徑寫成資料，不要寫成探索器。** 依原版手冊、Adventurer's Journal、
> City Hall 委任 producer 清冊（spec 024／028／029／038、`cmd/pool-city-hall-audit`）與
> 港務長機制（spec 102），列出「從開場到 Tyranthraxus」每一個必經委任、它的完成旗標、
> 進入該區的正常入口、離開該區的正常出口（牆面、事件選單或船），並逐條標證據等級
> （`exact`／`strong inference`／`hypothesis`）。出口不明的區（GEO1/31 的 ECL1 block 24、
> ECL 18 → 29／32 → 返城）用 `cmd/pool-ecl-trace` 讀原版分支，必要時以 dosgolem 走一次
> 原版看它怎麼離開；**不准用「再多探索一會」代替查證**。這張表放進 `docs/spec/` 或
> `docs/playtest/`，是後續導航的唯一依據。
>
> **第 2 步：把測試駕駛改成委任狀態機。** 每一段「目前委任狀態 → 目標區 → 已知出口」
> 是一個有界步驟，每個迴圈帶 guard；guard 給寬（它的用途是讓迴圈有終點，不是限制
> 次數），但撞到 guard 要 `t.Fatalf` 印出當下 ECL／GEO／座標／朝向／關鍵旗標
> （`4ABA 4AC1 4AA7 4A01 4A06 4A26 4A9E`），不能靜默繼續。把跨整個世界的隨機探索從
> 主線測試拿掉；探索器只留給荒野口袋那種原版本來就靠隨機事件的段落。
>
> **第 3 步：卡住就縮小，不要重跑整條。** 同一段第二次以同樣形狀失敗，先寫 0.x 秒的
> 最小重現（從已知狀態直接進該段，只證明那一段），修好再回整條。整條測試每跑一次
> 約 5 分鐘，**開跑前先講「這一輪要回答的問題跟上一輪差在哪」**，答不出來就不跑。
> 不得靠加大 `-timeout` 讓它過。
>
> **第 4 步：留收據。** 通過後把走過的 map／block 序列、委任槽終值、Tyranthraxus 戰鬥
> 進入證據與 PROGRAM 8 結局頁數寫成測試斷言，並在 `docs/playtest/` 留一份人讀得懂的
> 經過紀錄（每一段：進入狀態、按鍵摘要、離開狀態）。
>
> **不可越線**：不用 direct-entry、座標／旗標注入、forced-win、資源或能力值強化；
> 真實全滅是合法結果——遇到就停下記錄，不改隊伍強度硬過。不改共用 engine 來放
> Pool 專屬邏輯；不碰 CoAB。測試治具的輸入行為要與真實 `Update()` 輸入層一致。
> 建置與測試走 `tools/go.sh`（Docker），主機只做 git 與編輯；`fmt` 不動既有檔案。
>
> **台帳同步**：每一輪結束用 host `gh` 在 #5 留言：跑了什麼指令、走到哪、新的失敗
> 形狀、下一個最小工作。只有 `TestMainlineProbeNaturalPartyFirstBattle` 從標題到結局
> 不中斷通過、且收據已寫入時才關閉 #5；關閉後再改 `docs/worklist.json`、
> `go run ./cmd/pool-worklist -mode render -write WORKLIST.md`、`-mode verify`，最後
> `gh issue list --state open` 盤點，本地發現的新工作一律先開 issue。
>
> **停止線**：同一段以同一成因失敗第三次，或發現要「改遊戲規則讓它過」，停下來回報，
> 不再嘗試。

## 已知風險與待決

- 主線必經委任的**完整清單**目前分散在多份 spec 與 issue 留言裡，沒有單一表；
  第 1 步做出來之前，任何「還差幾段」的估計都是猜的。
- 港務長的兩道閘門（朝北、手上無票）與 `4AA7`／`4A01` 順序限制（spec 102）是第二輪
  循環的可能成因之一，但**尚未查證**；第 1 步要先確認狀態機在該處的前置條件。
- 荒野隨機事件依賴 `eclSeed`／`roller` 種子；種子改了路徑會變，收據要記種子。
