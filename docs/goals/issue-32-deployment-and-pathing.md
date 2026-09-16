# Goal：戰場的部署與走位對回原版（GitHub #32；spec 053／057／061）

主台帳：[issue #32](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/32)。
上一個 goal 的結案位置：[`issue-22-tactics.md`](issue-22-tactics.md)（#22 留著；三個變因沒有一面牆過，
成因不在駕駛，`docs/playtest/mainline-end-to-end.md` 補七）。

## 現況（2026-09-16，commit `29da999` 已推送）

- 貧民窟三場固定事件（24／34／33）與索寇 (8,5) 開打那一刻的盤面已經印得出來
  （`boardLines`、`TestWallSlums*` 的 `onBattleStart`）：隊伍一律一橫排、敵人右邊三到五排，
  法師在前排；戰場是兩條斜牆夾的斜街。這個形狀是 remake 的暫時部署擺的
  （`provisionalRoster`：隊伍 −1、敵方 ＋2，spec 061「remake 的暫時部署」一節），原版的
  樣板填寫者與 `DS:45BAh` 還沒讀。
- 衛兵那兩場的僵局：敵方在斜街 1 格寬處「CLOSED 0 STEPS」六十回合，最後靠**非原版**的
  僵局安全閥（`tacticalStalemateRounds = 50`）收場，經驗值與計數都沒有。
- 已知的 exact：移動預算與八方向成本（spec 053）、目的格 probe（spec 058）、佔用格重建
  `03A2h` 與部署 `14CFh`（spec 061）、敵方五方向表 `02ACh` 與換模式（`tactic_offsets.go`）。
  DRAFT：`2759h`／`275Ah` 兩欄的玩家語意、`45BAh` 的寫入者、樣板組數、斜向走位。
- 工具：`tools/ida.sh`（overlay-10／13／31／32）、`docs/audit/ida-overlay*.json` 的匯出腳本、
  dosgolem（`tools/dosgolem-reference.sh`，可送鍵、可讀變數）。

## 2026-09-16 停在哪

step 1 讀完（spec 061 CONFORMED、057／053／078 補齊）、step 3 換完（`combat.PlaceCombatant`、
遭遇距離走法、COMBAT 開打前壓距離）、三場兩 seed 進補八；dosgolem 兩筆收據（單人隊、
五人隊）開打九格逐格相同。獸人家那一場先走不到（鎖著的屋子、走路遭遇讓盲送鍵作廢），照停止線開了 #33，
接著把它修好：`shots` 加狀態檔、`tools/dosgolem-drive-orc-home.py` 看畫面再送鍵，
三場固定事件都到了（25／37／28 格逐格相同），#33 關。走位量到 remake 多的「要離目標
更近」與繞路兩條非原版備案把後排鎖住，拿掉後照 spec 096 走；誰先動仍看骰。dosgolem 加逐步快照
（`-trace-peek`）之後對到動作層：三場 40 個原版動作 remake 重現 32 個
（`TestFoeWalkReproducesEveryOriginalAction`），卡住第二次改照原版重挑再走；再加 `-trace-call`
把骰流讀出來餵回去，40 個動作全部停在原版那一格——途中抓出三條規則差（步的成本門檻、剩 1 點
收工、全不通換模式再試）與候選名單的排序／兩輪制，都照原版改了（spec 096 CONFORMED）。
索寇 GEO4/21 那一場重跑不再隔牆（安全閥 0 次），Sokal 探索測試變慢是紅的副作用（十個種子
跑滿，成因在 #22）。#32 關；戰術層剩下的唯一非原版備案是僵局安全閥，有觸發計數。

## 提示詞（可直接貼給 `/goal`）

> 目標：把戰場的部署與走位對回原版，讓貧民窟三場固定事件的開打陣型與敵方走位跟原版
> 同狀態一致；主台帳 #32。
>
> 1. **先讀原版，不改 remake。** 三支要讀：overlay-10 `14CFh` 的呼叫端（誰算地城格偏移、
>    誰寫 `DS:45BAh`、樣板組幾個）；overlay-31 路徑候選對 `2759h`／`275Ah` 的用法與斜向那
>    一步的判定（兩側正交格是牆時走不走）；overlay-32 `03A2h` 佔用格重建對 `+10Ch` 各狀態
>    （4 昏迷、5 倒地、6 死、睡著的效果）的處理。每一條標位址、bytes、證據等級，寫進
>    spec 061／053／057；讀不到的標 DRAFT 並寫明卡在哪。
> 2. **拿原版當裁判。** dosgolem 從貧民窟入口走到獸人家（地形 9）開打，讀 `DS:6039h` 佔用格
>    與 combatant 位置表，記下開打陣型（隊伍六格、敵方二十四格）；再讓敵方走三回合，記每
>    回合的位置。remake 同狀態（同隊伍、同格、同朝向）印同一份，兩張表對照。
> 3. **改 remake。** 部署照讀出來的偏移與樣板換掉 `provisionalRoster`；走位照 `2759h`／
>    `275Ah` 的規則；佔用格照 `03A2h`。原版沒有的（繞路備案、僵局安全閥）標成 remake-owned，
>    留著的要有觸發計數與理由。一次改一項，每項改完跑 `TestWallSlums*` 三場兩個 seed 看
>    陣型與結果，表接進 playtest 補八。
> 4. **不可越線：** 不改能力值／裝備、不注入旗標、不 forced-win、不動 engine／CoAB；
>    `internal/combat` 只能照讀出來的原版規則改，不能為了讓駕駛贏而改。動到玩家看得到的
>    畫面（戰場陣型算）就跑 `tools/package-release.sh` ＋ `tools/appimage-dos-parity.sh`，
>    數字進 commit。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」**；先最小重現（一張固定盤面的部署、一步斜向走位）
>    再上三場；同 seed 對照。
> 6. **台帳：** 每輪在 #32 留言；spec 061／053／057 的狀態行更新；#22 若因此有一面牆過就在
>    #22／#5 留言。收尾 `docs/worklist.json` → `go run ./cmd/pool-worklist -mode render -write
>    WORKLIST.md` → `-mode verify` → `gh issue list --state open`。
> 7. **停止線：** 樣板填寫者三支 overlay 追完還讀不到；dosgolem 走不到獸人家（先修它，另開
>    issue）；改完部署三場結果沒有一場變、也沒有更像原版；一條測試超過 60 秒。

## 已知風險與待決

- 陣型樣板「不在檔案裡」（spec 061）：是執行期填的，填寫者可能在 overlay-10 之外，要跨
  overlay 追。
- 原版的敵方走位在斜街可能一樣卡（那就不是 remake 的問題），但原版沒有 50 回合安全閥——
  要看原版怎麼收場（玩家只能等或逃？）。
- dosgolem 對戰鬥畫面的送鍵與變數讀取（`docs/audit/dos-parity-sample.md` 戰鬥段 78%）
  夠不夠讀部署表，先試一次再排時程。
- #31（諾里斯 THAC0）與這一條無關但同在戰鬥數值層，順序上先做 #31（小）。
