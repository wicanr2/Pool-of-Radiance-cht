# Spec 052：Pool 戰鬥先攻分數與行動者選取

狀態：CONFORMED（DEX modifier、分數合成、驚訝修正、最大值與同值決勝、
施法時間、Delay、攻擊槽完成 primitive）；DRAFT（移動／防禦的先攻值消耗、
玩家／AI 行動與完整回合 continuation）。
日期：2026-09-01。

## 證據

- `overlay-13.bin` SHA-256：
  `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390`。
- `overlay-08.bin` SHA-256：
  `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f`。
- IDA Pro 9.4、overlay-local file offset、base 0、16-bit metapc：
  `docs/audit/ida-overlay13-attack-slot-init.json` 保存 overlay-13 entry 1 的
  `0084h..0107h`；`docs/audit/ida-overlay08-initiative-selection.json` 保存
  overlay-08 `0124h..01E1h`；`docs/audit/ida-overlay08-combat-loop.json` 保存
  `0071h..0123h` 的呼叫端。
- `docs/audit/ida-overlay25-dexterity-initiative-modifier.json` 保存 overlay-25 entry 11
  `1173h..1200h`；`docs/audit/ida-overlay13-casting-time-initiative.json` 保存 overlay-13
  entry 19 `23F9h..2555h`。
- `docs/audit/ida-overlay08-delay-command.json` 保存 overlay-08 entry 9 `0E2Fh..0FDAh`。
- `docs/audit/ida-overlay13-attack-initiative-finish.json` 保存 overlay-13
  `1404h..17F2h`；`docs/audit/ida-overlay13-attack-wrapper.json` 保存 `1883h..1B12h`。

## 原版資料流（exact）

1. overlay-13 entry 1 只對 record `+10Dh != 0` 的 combatant 建立 runtime `+3`。
   原始 far call `010A:0057` 依 MZ header `3B0h` 與 TPOV control table精確映射到
   overlay-25 entry 11。該 entry 讀 record `+13h`；Spec 004 已證實它是 DEX。
2. DEX modifier 表是：0..2=`-4`；3..5=`DEX-6`；6..15=`0`；16..18=`DEX-15`；
   19..20=`3`；21..23=`4`；24..25=`5`；其他值走原版 default `0`。
   entry 1 將這個 signed byte 加 `1d6`。
3. 相加結果小於 1 時先夾到 1。若 `(record[10Eh]+1) & global[596h]` 非零，之後才減 6。
   最終值小於 0 或大於 20 時寫 0；0 本身合法保留。因此順序不可改成「先減 6 再夾值」。
4. overlay-08 `0124h` 沿 `DS:5CF4h` 與 record `+104h` 遍歷 combatant；每筆先擲
   `1d100`，再比較 runtime `+3`。較大的 `+3` 取代目前候選。
5. `+3` 相同時，新的 `1d100` 大於或等於目前候選的決勝值才取代；所以完全同值時
   後出現者取代前者。最大 `+3` 為 0 時回傳空指標。
6. overlay-08 `0071h` 主迴圈反覆呼叫選取函式與 `01E4h` 行動處理；選不到角色才進
   `0868h` 回合邊界。這足以把 runtime `+3` 命名為「先攻排序分數」，但尚不足以宣稱
   每種行動如何消耗／重設該值。
7. overlay-13 entry 19 的施法選擇路徑從 spell table 取得一個 byte，除以 3 得到
   casting cost。若目前 runtime `+3` 大於 cost 就相減；否則寫 1。這只閉合施法時間
   對先攻值的更新，不把 spell table 原始欄位名稱或其他行動消耗一併猜出來。
8. overlay-08 entry 9 的原始 combat command menu 在選到字元 `D` 時，直接把目前角色
   runtime `+3` 寫成 1，並把完成旗標寫 1；這是 Delay 命令的 exact 玩家可見邊界。
9. overlay-13 `1404h` 的 `arg_A` 是 attacker：函式依它的 runtime attack selector
   讀 `+113h/+114h`、執行攻擊，最後由 slot 1 掃到 slot 2。任一 remaining count
   大於 0 就把完成旗標改回 0；兩槽皆為 0 才對同一 `arg_A` 呼叫 overlay-25 entry 34。
10. entry 34 同時把 runtime `+3/+0/+7/+6` 清零，不能只命名為 initiative setter。
    `1883h` wrapper 另把 `arg_A` target pointer 寫入 `arg_E` runtime `+0Ah/+0Ch`，再於
    完成旗標成立時清除 `arg_E`，交叉證明被清除的是 attacker，而不是 target。

## Typed primitives

`DexterityInitiativeModifier(dexterity)`：逐段重現上述原版表，包含 `26..255 → 0` 的
default；不要以現代 AD&D 表格擴張原版程式沒有的值。

`ResolveInitiativeScore(modifier, die, surprised)`：

- `modifier` 是 `DexterityInitiativeModifier` 的 signed byte 結果；
- `die` 只接受 1..6，其他值失敗即關閉；
- 依原版順序執行相加、minimum-one、驚訝減 6，再把 `<0` 或 `>20` 轉成 0。

`SelectInitiativeActor(scores, tieRolls)`：

- 兩陣列長度必須相同，每筆決勝值必須是 1..100，分數必須是 0..20；
- 選最大非零分數；同分以較大 `1d100` 取代，完全同值時後者取代；
- 全零回傳「沒有行動者」，不得自行挑第一名或跳到勝利。

`ApplyCastingTimeInitiative(score, castingCost)`：score 只接受 0..20；`score > cost`
時相減，否則回傳 1。primitive 不負責查 spell table，也不代表施法已完成。

`DelayInitiative()`：固定回傳原版命令寫入值 1；caller 必須先確認玩家確實選到 `D`。

`InitiativeAfterAttackSlots(score, primaryRemaining, secondaryRemaining)`：任一槽仍有攻擊時
保留 score；兩槽都為 0 時回傳 0。這只投影 entry 34 對 initiative 的效果；完整戰術
runtime 還必須同步清除原版其他欄位，不能只更新這個 byte 後宣稱 attack action 完成。

本規格刻意不接前端戰術畫面。其他行動的先攻值消耗與玩家／AI 行動尚未 READY；
這些缺口未閉合前，Slums staging 必須維持失敗即關閉，不能自動結算戰鬥。

## 驗收

- 固定普通相加、minimum-one 後驚訝、驚訝後恰為 0、上溢到 21 轉 0。
- 固定 DEX 表每段端點與原版範圍外 default。
- 固定最大分數、同分較大／相等／較小決勝值、全零與非法輸入。
- 固定 casting cost 小於、等於及大於目前 score 的三條路徑。
- 固定 Delay 命令結果為 1。
- 固定 primary／secondary 任一未耗盡時保留先攻，兩槽皆零才清除。
- Pool 全套 `go test ./...` 與 `go vet ./...`。
