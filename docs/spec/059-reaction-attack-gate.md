# Spec 059：離開威脅區的反應攻擊閘門

狀態：READY（觸發條件、朝向窗、攻擊槽選擇、致能效果碼表、效果串列走訪）；
DRAFT（效果系統的否決查詢 `overlay-13 entry 11`、`+108h` 結構的 `+3`／`+0Fh`
旁路旗標、`overlay-25 entry 27` 兩個查詢碼 `4Bh`／`4Ah` 的玩家語意）。
日期：2026-09-02。

## 為什麼需要這一段

spec 053 只證實「移動後可由附近敵方消耗一次性狀態發動反應攻擊」，並把
`+7` producer 與兩個 predicate 列為未閉合。本規格把整個閘門讀完，
並修正觸發條件：它不是「所有鄰接的敵人」，而是「因這一步而失去鄰接的敵人」。

## 證據

| 檔案 | 內容 | 輸入 SHA-256 |
|---|---|---|
| `docs/audit/ida-overlay13-opportunity-attack.json` | overlay-13 entry 6 `08B9h..0C51h`（331 條） | `4d53df20…2390` |
| `docs/audit/ida-overlay25-reaction-predicates.json` | overlay-25 entry 6 `0B79h`、entry 27 `21DCh` | `9fede24b…50c0e` |
| `docs/audit/ida-overlay13-reaction-predicate.json` | overlay-13 entry 11 `1087h..1130h` | `4d53df20…2390` |
| `docs/audit/ida-ds-disabling-effect-codes.json` | `DS:287Fh` 起 32 bytes | `12811cbc…10d9f` |

## 觸發條件是差集，不是鄰接

overlay-13 entry 6（`procedure(direction: byte; mover: far ptr)`，`retf 6`）：

1. 以 mover 與距離參數 1 呼叫 overlay-25 entry 32，把 `DS:6CD7h` 的結果
   （移動前鄰接的敵方索引）抄進區域陣列；一筆都沒有就直接結束。
2. 把 mover 在位置表（`DS:5E89h`）的座標**暫時**沿 direction 推一格，
   再查一次 overlay-25 entry 32，然後把座標復原。
3. 兩份名單相減：移動後仍在名單裡的，在區域陣列裡覆蓋成 0。
   原版不壓縮陣列，只是把不符合的槽清零。
4. 對每個剩下的索引跑下一節的閘門。

## 逐個對手的閘門

依原版順序，任一不成立就換下一個對手：

1. mover 的 record `+10Dh` 不為 0。
2. **overlay-25 entry 6(對手) 必須回 0**。它是「對手身上有沒有致能效果」，
   有就不打。
3. **overlay-13 entry 11(mover, 對手) 必須回非 0**。
4. **overlay-25 entry 27(對手, `4Bh`, …) 必須回 0**。
5. **overlay-25 entry 27(對手, `4Ah`, …) 必須回 0**。
6. 以對手 `+108h` 結構的 `+9`（目前朝向）為基準，跑五個朝向
   `(base+6) mod 8` 到 `(base+10) mod 8`——也就是目前朝向的前後兩格。
   對每個朝向：已經打過就跳出；`+108h` 結構的 `+3` 大於 0，或 `+0Fh` 為 0，
   就直接視為成立；否則以 overlay-31 `0579h`（spec 056 的朝向弧）判定
   mover 是否落在對手該朝向的弧內。
7. 成立就選攻擊槽、發動一次攻擊，並把「已打過」旗標設起，該對手不再重試。

## 效果串列與致能效果碼

overlay-25 entry 27（`21DCh`，`retf 0Ah`）是效果串列的線性搜尋：
串列頭在 combatant record 的 `+7Fh`，每個節點的 `+0` 是效果代碼、
`+5` 是下一個節點的遠指標。找到指定代碼就回 1。

overlay-25 entry 6（`0B79h`，`retf 4`）以 `DS:287Fh` 為折疊基底，
逐一用索引 1..4 取出四個代碼問 entry 27，任一命中就回 1：

| 索引 | 代碼 |
|---|---|
| 1 | `33h` |
| 2 | `34h` |
| 3 | `35h` |
| 4 | `1Fh` |

這四個 byte 在 `2880h..2883h`，緊接在佔格偏移表的結尾 `287Fh` 之後。
代碼的玩家語意（哪一種失能狀態）尚未閉合，不得先命名。

## 攻擊槽選擇

1. 對手 record 的 `+A1h`（主攻擊 base rate，spec 051）非 0 時預設用第一槽，
   否則用第二槽。
2. 依序看兩個 phase 計數 `+113h`、`+114h`，計數大於 0 的槽覆蓋預設，
   後看到的贏。
3. 選中的槽計數若為 0 就補成 1。
4. 把槽號寫進 `+108h` 結構的 `+4`。

## overlay-13 entry 11

`function(mover, opponent: far ptr): boolean`，`retf 8`。mover 為 NULL 回 0；
mover 與 opponent 相同回 1。其餘情形把 `DS:677Ch` 當否決旗標清零，
呼叫 overlay-24 entry 3（效果分派）兩次——一次針對 mover、一次把 opponent 的
`+108h` 結構 `+0Ah` 暫時指向 mover 之後針對 opponent，事後復原——
最後以 `DS:677Ch` 是否仍為 0 決定回傳值。

因此它是「效果系統有沒有否決這一次攻擊」的查詢。哪些效果會設 `677Ch`
尚未閉合，所以本規格不宣稱它等同任何具名規則。

## READY 契約

1. 反應攻擊的候選是差集，不是全部鄰接敵人；差集要在「暫時移動、查詢、復原」
   之後計算。
2. 五個朝向的順序是 `(base+6)..(base+10) mod 8`，不可改成以 mover 方位直接算。
3. 每個對手最多打一次；已打過的旗標要在朝向迴圈內就生效。
4. 四個致能效果碼照 `DS:2880h` 原樣搬，不得改名或增刪。
5. 效果串列以 `+7Fh` 為頭、`+5` 為 next 走訪，順序影響 entry 27 的結果，
   不可改成集合查詢之外的順序假設。
6. 在 `677Ch` 的 producer 閉合前，effect 否決查詢一律保守處理，
   不得預設為通過。
