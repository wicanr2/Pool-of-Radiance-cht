# Goal：測試駕駛的玩家策略層——讓自然隊伍活過貧民窟（GitHub #22、#25）

主台帳：[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（策略層）與
[issue #25](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/25)（BANDAGE）。
本文件是給下一個 agent session 的開工提示詞；進度、證據與關閉狀態一律回到 issue。
上一個 goal（#5）的結案位置：[`issue-5-mainline-end-to-end.md`](issue-5-mainline-end-to-end.md)。

## 現況（2026-09-15，commit `7435e55` 已推送）

路線層從標題到結局的狀態機已接通（spec 137、`cmd/pool-game/mainline_castle_test.go`），
用注入的隊伍 1 秒走完。自然建角的隊伍（重擲到力量 ≥ 17、HP ≥ 9，開場買甲盾劍並裝上）
在貧民窟第 7 場全滅（`4ABB = 07`），經過在
[`docs/playtest/mainline-end-to-end.md`](../playtest/mainline-end-to-end.md)。死因有證據：

- 戰鬥拖到 15～21 回合，倒地（狀態 5）的人九回合後轉死亡（`combat.AdvanceDyingCounter`，
  overlay-08 `0868h`），remake 沒有 BANDAGE（#25）；死亡要 5500 GP，一級隊伍付不起。
- 戰術駕駛 `tacticalPilot.key`（`coverage_test.go`）只會逐格衝向最近的敵人、相鄰就 A。
- 沒有訓練升級（spec 097 已實作 `trainMember`，駕駛沒去用）、沒有神殿治療（spec 115 已實作）。

四條探索器測試同樣紅（#22 列了名單），其中 `TestRandomWalk…` 另有一個與 HP 無關的硬失敗
（`ecl8/29` `0x2D` 選擇子 `0x10`）。

## 提示詞（可直接貼給 `/goal`）

> 目標：讓 `TestMainlineProbeNaturalPartyFirstBattle` 的自然隊伍以正常按鍵活過貧民窟 25 場
> （`4ABB == FEh`）並交件（`4ABB == FFh`），主台帳 #22／#25；四條探索器測試回綠。
>
> 1. **先做 #25 BANDAGE，它是死亡的直接成因。** 從 overlay-08 的戰鬥指令分派
>    （spec 129 的 `Quick Done` 段，說明書 p.41 DONE 底下 GUARD／DELAY／QUIT／BANDAGE）讀出
>    入口、條件（相鄰、目標狀態 5）與效果（狀態 5 → 4？計數歸零？）寫成 spec，證據等級逐條標；
>    讀不出的用 dosgolem 走一次原版看畫面與記錄。接進戰術輸入層（從 `Update()` 送鍵），
>    寫一條 0.x 秒的單元測試證明「倒地的人被包紮後撐過第 10 回合」。
> 2. **戰術駕駛改成有優先序的規則，不改規則本身：** 有倒地隊友且相鄰 → BANDAGE；
>    HP 低於某比例且相鄰敵人 → 退一格或 GUARD；否則集中打「已受傷且最近」的那一隻。
>    規則只動 `tacticalPilot`（測試治具），不動 `internal/combat` 的判定。
> 3. **戰後策略：** 每場打完照順序判斷——有人死亡且付得起 → 神殿 Raise Dead；
>    有人 HP < 1/2 → 紮營休息（`E R Y I×天 R`，一天回 1 點，#24 的打斷先照現況）；
>    任一人經驗值過門檻 → 走到訓練所 `trainMember`（spec 097，付 1000 GP 的規則照原版）；
>    金幣夠 → 回武具店升級甲（`outfitShoppingList`）。每一段是有界步驟，撞 guard 就 `Fatalf`
>    印 `4ABB` 與六人的 HP／狀態／XP／金幣。
> 4. **每一輪開跑前寫出「這輪跟上輪差在哪」**，並先跑最小重現（單場戰鬥、單次休息），
>    整條探針（約 4～5 秒）只用來量結果。結果表接到 `docs/playtest/mainline-end-to-end.md`：
>    第幾場、誰死、等級、AC、HP。不准加大 timeout。
> 5. **`TestRandomWalk…` 的 `0x10` 選擇子硬失敗另查成因**，與 HP 無關，查到就開 issue 或修。
> 6. **不可越線：** 不注入旗標／座標、不 forced-win、不改能力值或金幣、不改 engine／CoAB、
>    不為了讓測試過而改判定規則（發現 remake 規則與原版不符 → 先開 issue 帶證據再改）。
>    全滅仍是合法結果，記錄它在哪一場、什麼等級。建置測試走 `tools/go.sh`，`fmt` 不動既有檔。
> 7. **台帳：** 每輪在 #22 留言（指令、走到第幾場、死因、下一個最小工作）；BANDAGE 接上並有
>    測試後關 #25；25 場＋交件通過後關 #22，接著回 #5 續跑索寇要塞以後的段落。關閉後同步
>    `docs/worklist.json` → `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` →
>    `-mode verify` → `gh issue list --state open` 盤點。
> 8. **停止線：** 同一場、同一死因第三次；或發現要動 `internal/combat` 的判定才過得去；
>    或一條探針超過 60 秒——停下來回報。

## 已知風險與待決

- BANDAGE 在原版的效果只有說明書一句話（p.41「包紮裹傷」），狀態轉換與計數歸零要從
  overlay-08 讀；讀不出來就標 hypothesis 並用 dosgolem 實測一次。
- 訓練所升級的門檻（spec 071）對一級隊伍是 2000 XP 上下，貧民窟 25 場夠不夠升到二級
  沒有量過；探針要把每場的 XP 累計印出來。
- 預設建角是五戰士一賊，沒有牧師——第一級沒有治療法術，休息是唯一回血手段。
  若量到「不換職業組合就過不了」，那是建角策略的問題，回報後由使用者決定，不自行改建角。
