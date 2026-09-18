# Goal：讓測試套件回到「紅就是壞了」——任意鍵（#54）＋探索器的治療策略（#22），外加兩筆原版量測（#21、#39）

主台帳：[issue #54](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/54)（全滅畫面寫著 Press any key，
但方向鍵不算）、[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（探索器測試的玩家策略層：
休息／治療／訓練）、[issue #21](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/21)（腳本搬座標之後
remake 還會再走原本那一步）、[issue #39](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/39)（城區晚上鎖門）。
上一個 goal 的結案位置：[`issue-19-22-23-red-tests-and-original-receipts.md`](issue-19-22-23-red-tests-and-original-receipts.md)〈2026-09-18 收在哪〉。
主線驗證已由使用者 2026-09-18 判定足夠，#19／#45／#46 因此關閉。

## 這一輪要換掉什麼狀態

`go test ./...` 現在固定 **七條紅**，而且七條**都是同一個成因**：探索器與探針的一級隊伍在城區撐不到走完，
每一趟都全滅。紅燈長期擺著的代價不是「有七條壞掉」，是**下一個人分不出「本來就紅」與「剛弄壞」**
——這一輪的目標是讓「紅」重新代表「壞了」。

順序固定 **#54 → #22**：兩者是同一段工作的兩半。修 #54（方向鍵也算任意鍵）之後，全滅會真的回標題，
而 `returnToTitleAfterGameOver` 會把 session 拆掉；治具有 **141 處**直接呼叫
`application.eventSession.CurrentBlockID()`，上一輪實測有兩條測試當場 nil panic。
所以 #54 不能單獨出，它的收尾就是 #22 的治具整理。

#21、#39 是兩筆便宜、獨立、有明確驗收條件的原版量測，放在同一輪當作 dosgolem 的熱身
（上一輪剛寫了兩支駕駛：`tools/dosgolem-tour-cell-rest.py`、`tools/dosgolem-party-wipe.py`，照抄就能開工）。

## 現況

**#54（產品缺陷）**

`anyKeyJustPressed`（`cmd/pool-game/main.go`）只認 `A`..`Z`、`0`..`9`、ENTER、SPACE、ESC。
畫面上寫的是 `Press any key to continue`。探索器因此在全滅之後按方向鍵**十八萬圈原地不動**。
原版那一頁吃不吃方向鍵**目前量不到**（dosgolem 推不動，#53）——在那之前以畫面上的字為準。

**#22（策略層）**

上一輪讓全滅在探索器裡成為這一趟的終點（結束理由寫成「全滅於 GEOx/y (a,b)」，
世界巡迴 80 秒 → 10 秒，整包 190 秒 → 96 秒），於是每一趟的結局一目了然：**每一趟都全滅**，
位置散在城區各處（(10,5) 神殿守衛、(10,8) 扒手、(15,10) 城衛隊）。

已知會影響策略的兩件事：

1. **城區街上休息會被城衛隊攔**（原版就是這樣：排兩小時、00:05 打斷、`GO`／`STAY`，`STAY` 會開打；
   收據 `docs/audit/dos-tour-cell-rest.json`）。所以「原地紮營」不是治療策略。
2. CLAUDE.md §6：真實全滅是合法結果，**不強化隊伍**。所以是讓探索器像玩家一樣回頭補血
   （神殿、旅店、貧民窟的房間），不是給它鎖 HP。

上一輪試過但退掉的做法：把會開打的選項（`STAY`／`FORCE YOUR WAY PAST`／`GRAB`）一律改選被動的那一個
——**打地鼠**，每修一組就換下一格全滅，而且會改到別條測試的路線（索寇碼頭的 `STAY` 是主線要的）。

**#21（落點差一格）**

`ecl5/5 A467h`「YOU GO UPSTAIRS.」寫 `SAVE 5 → C04B`、`SAVE 7 → C04C` 再 `NEWECL 7`。
remake 套用座標之後，`moveInitialDungeonForward` 又把玩家原本按的那一步加上去，落點成了 (6,7)。
原版是「先走再跑入口 0」還是「入口 0 先跑、腳本沒搬人才走」沒有實跑證據。
**影響所有 `SAVE → C04B/C04C` 後接 `NEWECL`／`EXIT` 的傳送。**

**#39（晚上鎖門）**

spec 102 的靜態分支是 exact、比較方向已用 dosgolem 對過；缺的是 runtime 收據
（原版晚上站上鎖門格的畫面）與「牆型 9 的十一個門面對到哪幾棟」。

## 提示詞（可直接貼給 `/goal`）

> 目標：七條紅燈變成「紅就是壞了」。做完關 #54、#21、#39；#22 依結果關閉或把剩下的縮成新 issue。
>
> **A. #54 任意鍵＋治具的收尾（一起做，不能只做一半）**
>
> 1. `anyKeyJustPressed` 改成真的認任何鍵。
> 2. 治具的原則是「**回標題了就這一趟結束**」，不是逐處補 nil 判斷——141 處呼叫不可能逐一改對。
>    先找出真正會走到全滅的那幾條測試（上一輪已知兩條：`TestTheCastleBehindStojanowGateHasContent`、
>    `TestTheWildernessCaveRerollReachesTheEasternOutpost`），在它們的迴圈頂端判 `gameOver`／`eventSession == nil`。
> 3. 跑全套，確認**沒有新的 nil panic**，紅燈數量不因這一步增加。
>
> **B. #22 治療／休息的策略層**
>
> 4. 先決定策略再寫碼，寫進測試的註解：什麼時候該回頭補血（HP 比例？有人倒地？），去哪裡補
>    （神殿要錢、旅店要錢、貧民窟的房間免費但有遭遇），補完怎麼回到原本的探索目標。
> 5. 實作在探索器與探針共用的那一層；**所有動作走正常按鍵**，不注入旗標或座標。
> 6. 驗收看三條探索器與三條探針：能綠就綠；**綠不了的要把預期改成「記錄停在哪」並寫明它現在證明什麼**
>    （走得到哪、停在哪一步），不要留一條看起來像通關證據的斷言。
> 7. `TestDirectedExplorationReachesMaps` 的「3 張地圖」是上一輪抓到的過期斷言
>    （把隊伍死後走到的也算進去），這一輪要給它一個有依據的新數字。
>
> **C. #21 上樓落點（dosgolem）**
>
> 8. 照 `tools/dosgolem-party-wipe.py` 的形狀寫一支：走到 `ecl5/5` 的樓梯、上樓、量落點座標與朝向。
>    收據寫 `docs/audit/`，記鍵序與狀態檔 SHA-256。
> 9. 與 remake 的 (6,7) 比；不同就改 `moveInitialDungeonForward`，並把 spec 137 第 10 段的 hypothesis 改成定論。
>    **改完要跑主線探針**——那一段是城堡的必經路線。
>
> **D. #39 晚上鎖門（dosgolem）**
>
> 10. 在原版把時鐘推過 14 點（紮營排小時，不注入），走到 (3,4) 朝東，收一張鎖門問句的畫面與 `6E7D = 8`；
>     同一路線白天要進得去市政廳（正對照）。
> 11. 十一個門面對到建築名（踏進去那一支印的第一句話，或 spec 102 的索引表），寫進 spec 102 並拿掉 OPEN 那一條。
>
> **E. 台帳與收尾**
>
> 12. #54、#21、#39 以證據關閉；#22 依結果關閉或縮成新 issue（**要有 GitHub issue**）。
>     `docs/worklist.json` 同步、重生 `WORKLIST.md`，再跑 `-mode verify` 與 `gh issue list --state open` 雙向核對。
> 13. 動到玩家看得到的畫面就打包後對拍；對拍會自己對基準表（`cmd/pool-parity-check`），數字寫進 commit message。
>     **#21 若改了落點就一定要對拍**——那會動到城堡那幾張。
>
> **F. 停止線**
>
> - B 的策略層做了兩輪仍然每趟全滅 → 停下，把「記錄停在哪」寫成正式預期，剩下的縮成新 issue。
> - #21 的原版量測走不到樓梯（要先清索寇要塞）→ 記下做過什麼，改用 `workplace/dosgolem-cheat/ending-castle.state` 之類的既有狀態檔接續。
> - #39 推時鐘推不過去（紮營被打斷，見上面的現況）→ 那本身就是答案的一部分，記下來再想別的推法（旅店？野外？）。
> - 要改 `internal/combat` 或共用 engine 才做得到 → 停下確認 CoAB 不受影響。
> - 單一步驟超過 30 分鐘。

## 已知風險與待決

- **A 與 B 的順序不能反**：先做 B 的策略層、隊伍不死了，#54 的症狀就不會出現，那一條會被誤判成「不影響」。
- **「記錄停在哪」不是放水**：要寫清楚它現在證明什麼，否則下一輪的人會把它當成通關證據
  （這一條上一輪已經踩過一次：`TestDirectedExplorationReachesMaps` 的 3 張地圖）。
- **#53 還開著**：原版全滅那一頁 dosgolem 推不動，所以「原版吃不吃方向鍵」這一輪仍然無法對照；
  #54 的依據是**畫面上自己寫的字**，這一點要寫進 spec 而不是假裝有原版證據。
- **#52（戰利品截圖）、#50（考古圖鑑）、#17（sprite 重繪）、#6（真機啟動）不在這一輪**。
  其中 **#6 是發行硬門檻**，建議排在這一輪之後。
