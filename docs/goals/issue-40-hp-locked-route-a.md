# Goal：HP 鎖定診斷通關——路線 (a) 從標題按鍵跑到結局（GitHub #40；#26、#22 → #5）

主台帳：[issue #40](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/40)（HP 鎖定診斷通關）。
相關：[#26](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/26)（路線 (a) 已決定）、
[#22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（正常強度的策略層）、
[#5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)（正常強度通關，維持 open）。
上一個 goal 的結案位置：[`issue-22-26-levels-before-sokal.md`](issue-22-26-levels-before-sokal.md)
（第 4、5 步量到停止線：索寇 (8,5) 19 個 seed 全滅、索寇之前撐等級的兩條路都沒走通）。

## 為什麼是「診斷收據」

使用者 2026-09-17 決定：走路線 (a)，並**鎖定 HP 強制通關**。CLAUDE.md §3／§6 規定主線驗收
「不依賴 forced-win、測試專用強化」「不用強化隊伍或 forced-win 強求通關」，所以這一條的收據定位是：

- 證明路線 (a)、旗標、換圖、交件、結局這一整條鏈，**從標題以正常按鍵走得到底**；
- 把路上每一個**不是戰力造成的**卡點（駕駛、remake 缺陷、原版規則讀錯）找出來修或開 issue；
- **不關 #5**。正常強度的牆（諾里斯約 15%、索寇 (8,5) 19 個 seed 全滅）留在 #22。

## 現況（2026-09-17，`f168b89` 已推送）

- open issues 10 條（#5 #6 #17 #19 #21 #22 #23 #26 #39 #40），`docs/worklist.json` 對得上。
- 全套測試紅燈五條：`TestSokalKeepOpensTheOtherBoatRoutes`、`TestWorldTourReachesTheAreasBehindTheHarbour`、
  `TestPlayingTheWorldCompletesCommissionsOnItsOwn`、兩條主線探針。
- 主線探針（`runMainlineProbe`，`cmd/pool-game/mainline_probe_test.go`）**已經涵蓋到結局**：貧民窟 → 交件 →
  （house rule）古托井諾里斯 → 交件、訓練、買甲 → 索寇要塞 → 東航線 → 波多廣場北緣 → 斯托亞諾夫城門 →
  城堡上樓 → 覲見廳 → 結局（`sailEast`／`crossToPodol`／`podolNorthEdge`／`stojanowGate`／`castleUpstairs`／
  `audienceHall`）。第 6～11 段只在注入的強隊上實跑過（spec 137），自然隊伍從沒走過索寇 (8,5)。
- 已有的治具：貧民窟就近進屋休息（`slumsIndoor`／`restIndoors`）、旅店、睡到早上、城衛隊問句照安全選項答
  （`cityWatchAnswer`）、索寇裡睡不成帶著現狀往下打（`pressOn`）、交件集中獎金、訓練所、隊伍 pool 買甲。
- 收據 seed：house rule 142。

### 路線 (a) 已讀出來的事實

**貧民窟 25 場**（`4ABB == FEh`，spec 042）：`ecl2/20 B69Ch` 是 `COMPARE @4ABB, 254 ; IF >= ; RETURN ; ADD 1`，
有 14 個 GOSUB 呼叫點、12 個固定事件（搬穀物袋三個變體）；隨機遭遇在 `4A80` 滿 15 之後停發（`9B32h`）。
所以 25 = 隨機 15 ＋ 固定 10。補八那 20 場打了 5 個固定事件，避開的是：

| 地形碼 | 呼叫點 | 事件 | 編成 |
|---:|---|---|---|
| 9 | `AA02h` | A LARGE ORC RAISES HIS HEAD（獸人的家）| 4×20、14×3、15×1 |
| 13 | `AB91h` | GUARDS RUN TO INTERCEPT YOU | 5×4、4×30 |
| 15 | `AC4Bh` | YOU HAVE ALERTED THE GUARDS | 3×21、2×12 |

其餘還沒確認打過沒有的固定事件：`A3B2h`（THE MAN TAKES THE POTION，4×8）、`ABFEh`（MONSTER LEADERS）、
`AF7Ah`（THE MAN RUNS SCREAMING，7×4、6×15）、`B10Dh`／`B134h`／`B14Fh`（搬穀物袋）、`A0BEh`（YOU PRESUME TOO MUCH，
36 隻那一場）。殺算命的老婦人（`A749h`）那條不是玩家會走的路，不走。

**波多廣場的委任**（spec 137〈波多廣場的委任要先接〉，exact 靜態）：職員清單從索引 0 起、跳過已完成、
一次最多三條（貧民窟、索寇、書、波多廣場……），貧民窟交件之後波多廣場排第三，列出時 `AA2Ah` 寫 `4AB0 = 1`。
拍賣在換圖載入時（入口 4 `ecl1/18 9971h`）看 `4AB0 == 1` 才開；進場選單 `9A36h` 選 DISGUISE PARTY AS
MONSTERS.（`@B255` 得 1），地形碼 1 → STAND AND LISTEN → WAIT FOR WINNER → `A60Ch` → `A865h`／`A86Bh` 寫 254。
喬裝在出價、靠近、離開那幾支（`A875h`）每次 1/5 被識破；WAIT FOR WINNER 那條不叫它。古托井西緣 → 波多廣場
是陸路（`TestKutoWellWestEdgeLeadsToPodolPlaza`）。

**HP 放在哪**：戰鬥中是 `tacticalState.HitPoints`（以 `PartySlot` 對回隊伍索引），戰後由 `storeCombatHitPoints`
寫回 `state.Party[i].CurrentHP`／`Status`（狀態 4 昏迷、5 瀕死、6 死亡，`internal/combat/round.go`）。

## 提示詞（可直接貼給 `/goal`）

> 目標：一條 HP 鎖定的主線探針，以 house rule（spec 140）走路線 (a)，從標題以正常按鍵跑到結局
> （泰倫斯拉克斯、`4ABA = FEh`、結局頁），收據寫進 playtest；主台帳 #40，完成後在 #26 改寫建議順序、
> 在 #5 留言。這是診斷收據，不關 #5、不關 #22。
>
> 0. **開工基準。** `git log -5`、`gh issue list --state open`、`pool-worklist -mode verify`；全套測試記紅燈
>    名單與耗時（上一輪五條，自己量）。
> 1. **HP 鎖定只做在測試治具，先寫最小重現。** 在 `cmd/pool-game` 的 `_test.go` 裡加一個鎖定器：每個 tick
>    把我方在 `tacticalState.HitPoints` 的值寫回 MaxHP、倒地／昏迷寫回 0，戰鬥外把 `state.Party` 的 HP 與狀態
>    寫回滿值；**每次寫回都計數**（戰鬥中、戰鬥外、從狀態 6 救回來的分開算，狀態 6 出現本身是鎖定的缺口，
>    要記位置）。先釘兩條 0.x 秒的測試：同一個 seed 的貧民窟衛兵攔截（34 隻）不鎖全滅、鎖了打贏且計數 > 0；
>    打完之後 `git diff --stat -- internal cmd/pool-game/*.go`（排除 `_test.go`）必須是空的。
>    **不改 `internal/combat` 與任何產品碼、不在發行包留開關、不注入旗標／座標／時鐘／金錢／經驗值、不改戰鬥結果**。
> 2. **貧民窟 25 場。** 鎖定模式下探針不再避開地形 9／13／15。先照〈現況〉的呼叫點清單，在 log 印出每一次
>    `4ABB` 加一是哪一個呼叫點（對 PC 或事件文字），補齊到 `FEh`；湊不滿時列出還沒打過的固定事件與它們的
>    觸發條件（逐支讀，標證據等級），不走殺老婦人那條。交件到 `4ABB == FFh`。
> 3. **職員列出波多廣場。** 交件後照正常按鍵聽完職員的清單，斷言 `4AB0 == 1`；印出清單上的三條。
>    沒有列出來就停下讀 `ecl3/8 A83Fh` 的迴圈，不要硬寫 `4AB0`。
> 4. **波多廣場拍賣。** 貧民窟 → 古托井西緣 → 波多廣場，進場選單答 DISGUISE PARTY AS MONSTERS.
>    （`walkToPodolPlaza` 的 entry 參數），走到地形碼 1，STAND AND LISTEN → WAIT FOR WINNER，斷言 `4AB0 == FEh`
>    而且全程沒有開打。把這一段抽成一條從鎖定探針狀態出發的收據（或存讀檔後的短測試），補上 spec 137 只有
>    靜態證據的那一半。回市政廳交件、house rule 折算、訓練所升級（照職業挑門）。
> 5. **諾里斯 → 索寇 → 第 5～11 段。** 照探針現有流程。每一段記入口／出口旗標（spec 137 表）、等級、XP、
>    時刻（`partyLine`）、HP 寫回次數、每場回合數。第 6～11 段第一次由自然建角的隊伍走，撞到的卡點：
>    駕駛問題就修治具；remake 與原版不符先開 issue 帶證據（spec／原版位元組／dosgolem），再決定修不修。
> 6. **收據與文件。** playtest 補十二（每段一列：旗標、等級、時刻、寫回次數、回合數、卡點）；spec 137
>    〈建議順序〉照 (a) 改寫並用實跑數字取代 hypothesis（鎖定下量的耗時與旗標可用，**勝負不可用**，要寫明）；
>    CONTEXT.md；本 goal 補〈收在哪〉。新測試綠，全套紅燈名單寫進 commit message。
> 7. **不可越線：** 見第 1 步的界線；不動 `golden-box-remake-engine` 與 CoAB；建置測試走 `tools/go.sh`，
>    `fmt` 不動既有檔；不准加大 go test 的全域 timeout（單條探針可給明確的 `-timeout`）；治具迴圈一律帶 guard；
>    用腳本改檔先驗錨點唯一再取代；暫時的 `t.Run` 掃描與追蹤碼收尾前刪掉。
> 8. **台帳：** 每段打通在 #40 留言（段落、旗標、卡點、下一個最小工作）；新發現的可執行工作先開 GitHub issue
>    再寫鏡像；跑到結局後關 #40，在 #26 更新建議順序、在 #5 留言「診斷通關完成、正常強度仍 open」；
>    鏡像把 #40 的 verify 從 manual 改成綁那條探針測試；`pool-worklist -mode render -write WORKLIST.md` →
>    `-mode verify` → `gh issue list --state open` 雙向核對。push 前先問。
> 9. **停止線：** 同一個非戰力卡點（同一段、同一原因）第三次；鎖定下出現無法結束的戰鬥（僵局安全閥收場、
>    敵方打不到或打不死）；要改 `internal/combat` 或產品碼才過得去；單一段超過 60 秒或整條探針超過 10 分鐘；
>    或發現需要注入旗標才走得下去——停下來回報。

## 已知風險與待決

- **鎖定的時機。** 一次 `Update()` 裡可能連續結算多下傷害，隊員在同一 tick 內就從滿血掉到死亡；每 tick 寫回
  補不到那一刻。狀態 6 出現就記下位置與那一 tick 的戰鬥紀錄，決定要不要把寫回挪到戰術盤每個行動之後。
- **貧民窟 25 場湊不湊得滿**：隨機遭遇 15 場的上限與固定事件的觸發條件都沒逐一實跑過；有些固定事件可能要
  特定朝向或前置旗標（`A3B2h` 的藥水、`AF7Ah` 的攤位）。
- **職員清單**的順序與「一次三條」是靜態讀出來的；交件之後有沒有別的旗標讓波多廣場排不上，第 3 步才知道。
- **第 6～11 段**只在注入強隊上跑過：城門索賄、城堡內部四張圖、覲見廳投票與衛兵，都是第一次接自然隊伍的狀態
  （`4A00`、`4A01`、時刻、金錢），spec 137 死區表列的那幾個狀態要逐一留意。
- **時刻**：市政廳晚上鎖門、城門的馬車商人 `49C9 >= 14` 不出現（spec 102／137），鎖定不處理時刻。
- 鎖定下的等級、耗時、旗標可以寫進路線表；**勝率不能**，那仍是 #22 的工作。
