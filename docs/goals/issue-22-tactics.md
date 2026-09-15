# Goal：駕駛的戰術層——三面牆同一成因（GitHub #22、#5）

主台帳：[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（策略層剩餘：戰術）、
[issue #5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)（主線收據）。
上一個 goal 的結案位置：[`issue-28-commission-experience.md`](issue-28-commission-experience.md)
（#28 已關；第二條探針停在索寇 (8,5)，`docs/playtest/mainline-end-to-end.md` 補六）。

## 現況（2026-09-15，commit `327fd22` 已推送）

- 兩條主線探針：原版規則（seed 136）停在索寇 (8,5)；house rule 開（seed 137）諾里斯交件
  升一級後也停在索寇 (8,5)。同一格同一死因第四次。
- 一級能完成的委任只有諾里斯（三個 seed 一勝兩敗），圖書館的書帶不出去（幽靈）。
  所以「靠委任升級再打大場」的前提是「先打得贏給委任的那幾場」——瓶頸回到駕駛。
- 三面牆的死法同一個形狀：敵人一擁而上、全隊散開各打各的、對 AC 4 打不中、包紮吃掉
  行動、睡著的敵人五回合後醒來沒人動它們。
- 駕駛在 `cmd/pool-game/tactical_pilot_test.go`（包紮→催眠→集火→BFS 接近）；主線駕駛
  在 `mainline_castle_test.go`／`mainline_house_rule_test.go`。四條探索器測試仍紅
  （它們用預設隊伍、沒接策略層）。
- 可用的原版機制：B）ANDAGE 不看距離（spec 138）、催眠（spec 098）、Q）UICK（spec 139）、
  D）ELAY／G）UARD（spec 062）、盤面地形 `tacticalState.Grid`（spec 061）。

## 提示詞（可直接貼給 `/goal`）

> 目標：讓駕駛在不改任何規則的前提下打贏一級隊伍「應該打得贏」的那幾場，量兩條主線探針
> 能走多遠；主台帳 #22、#5。
>
> 1. **先量再改。** 開工前把三場的回合紀錄拉出來（諾里斯 seed 136／138、貧民窟 24、索寇
>    (8,5)），每一場數：全隊命中率、花在包紮／移動／被擋的行動數、幾回合後睡著的醒來、
>    誰先死。結論寫成一張表，放進 `docs/playtest/mainline-end-to-end.md` 補七的開頭。
>    沒有這張表不准改駕駛。
> 2. **包紮的時機。** 身邊有醒著的敵人時不包紮，計時到 8 才包（spec 062：>9 才死）；
>    先用最小重現（`bandage_test.go` 那種 `Update()` 送鍵的）證明改法，再看三場的差。
> 3. **睡著的敵人要處理。** 沒有醒著的敵人在射程內時，去打睡著的（原版對睡著的沒有
>    自動命中，spec 138 附近的 overlay-13 讀出來的），優先打生命骰高的。
> 4. **站位。** 進戰場先看 `Grid` 有沒有窄處（門口、走廊）：有就法師牧師退到戰士後面、
>    戰士堵口；沒有就至少不散開——四個前排連成一線，法師牧師在後排。每一步都是 M 之後
>    的方向鍵（spec 139）。先在一條固定盤面的最小重現上驗（`newFoeTurnState` 那種），
>    再上探針。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」**；一次只改一個變因，同 seed 對照。三場各記
>    贏／輸、回合數、死幾個。同一場輸三次就停下換問題，不換 seed。
> 6. **不可越線：** 不動 `internal/combat` 的判定、不改能力值／裝備／金幣、不注入旗標、
>    不 forced-win、不動 engine／CoAB；house rule 之外不加規則。駕駛是測試治具，只能按鍵。
>    動到玩家看得到的畫面（不應該有）就要對拍。
> 7. **探索器那四條紅。** 改用 `mainlineDriver` 的策略層（`hurt`／`rest`／pilot），或把
>    預期改成「記錄全滅在哪」——選一種，理由寫進測試檔頭。`TestRandomWalk` 的
>    `0x10` 選擇子硬失敗另查成因，單獨一條測試釘住。
> 8. **台帳：** 每輪在 #22 留言（表＋差異）；兩條探針的段落表更新到補七；哪一面牆過了
>    就在 #5 留言。若量出來的成因是規則（例如命中判定與原版不同），開新 issue 標 exact／
>    hypothesis，不在這一條裡修。收尾 `docs/worklist.json` → `go run ./cmd/pool-worklist
>    -mode render -write WORKLIST.md` → `-mode verify` → `gh issue list --state open`。
> 9. **停止線：** 同一場同一死因第三次；要動 `internal/combat`；發現原版判定與 remake
>    不同（那是另一個 issue）；一條探針超過 60 秒；駕駛改到第五個變因還沒有一面牆過。

## 已知風險與待決

- 睡著的敵人「原版沒有自動命中」是 overlay-13 讀出來的（效果群組 16 沒有 34h／35h）；
  如果打睡著的照樣打不中 AC 4，第 3 條的收益就只剩「牠們醒不來」。
- 盤面的窄處要看 `tacticalState.Grid` 與部署位置（spec 061 部署上限）；索寇 (8,5) 那一場
  15＋4＋31 隻只站得下多少還沒對過原版。
- Q）UICK 交給電腦打不是選項——那是原版的 AI，不會比駕駛聰明。
- 四條探索器測試改預期會讓「探索器完成委任」那條收據變弱；改用策略層則每條要多跑
  幾十秒，先量再選。
