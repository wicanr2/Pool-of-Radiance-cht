# Spec 012：DOS cardinal coordinate wrapper 與 401Fh dispatch

狀態：READY（cardinal 座標更新、16×16 wrap、dispatcher call chain、1F40h ECL operand 唯一候選）；DRAFT（overlay-07 entry 27 座標用途、牆／門 collision gate、玩家輸入 mapping）
日期：2026-08-31

## 工具、輸入與位址空間

- IDA Pro 9.4，image `ida-pro-9.4-idapython:locked-v1`，metapc raw binary；
  `tools/ida-export-overlay-functions.py` 在分析後把 segment 設為 16-bit，保留原始函式
  seed、local offset、bytes、operands 與 IDA 版本，不用推測性改名取代定位。
- `overlay-07.bin` SHA-256：
  `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae`；
  位址空間為 overlay-local file offset、base 0。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`；
  位址空間同上。
- TPOV manifest 的零起算 overlay-07 entry 27：resident stub offset `00A7h`、code
  offset `1A17h`。overlay-03 全 corpus 直掃只找到一個 `9A A7 00 45 00` caller，位於
  `30FFh`。

## 401Fh dispatcher 與 1F40h operand 勘誤（READY）

overlay-03 entry 53（seed `3026h`）先由 resident operand resolver 取得 dispatch value。
`30FAh` 原始 bytes `3D 1F 40` 比對 `401Fh`；相等時 `30FFh` 原始 bytes
`9A A7 00 45 00` 呼叫 overlay-07 entry 27 的 resident stub。另一個相鄰值 `4019h`
在 `3106h` 走不同分支，只更新 view detail，不得和 movement wrapper 合併。

這證明 `401Fh → overlay-07 entry 27` 的 dispatch chain，推論等級為 `exact`。

2026-08-31 的舊工作假說曾把 `401Fh` 當成待尋找的 ECL opcode／`CALL` selector；
overlay-07 entry 2 與 entry 10 的資料流否定該說法。entry 2 把 operand 的 code、low、
high bytes 分別放入陣列，entry 10 以 `low << 8 | high` 組成 dispatcher 值。因此原始
ECL bytes `02 40 1F` 解碼為 code `02h`、word `1F40h`，再被 helper 反向組成 `401Fh`。

全八份 ECL 的逐位移候選掃描只找到一處 `02 40 1F`；以共用 decoder 驗證指令邊界後為：

- `ECL7.DAX` block 17；
- payload offset `1D9Ah`、address `B69Ah`（mapping base `9900h`）；
- opcode `27h` 的第四個 operand，word `1F40h`；record end `1DACh`。

此項把 producer 從「未知玩家命令」訂正成「opcode 27 operand resolver 的唯一候選」，
但尚未證明 opcode 27 在 Pool 的完整副作用，也未證明 `6A0Bh/6A0Ch` 是一般第一人稱
隊伍座標。不能因函式形狀像移動就把候選升格成玩家自由移動政策。

## overlay-07 entry 27 的座標運算（READY；座標用途 DRAFT）

函式範圍 `1A17h..1AA8h`，讀取朝向 `DS:6A0Dh`，只處理 cardinal `0/2/4/6`：

| facing | 原始範圍 | 座標更新 |
|---:|---|---|
| 0 | `1A1Fh..1A36h` | `Y > 0` 時 `Y--`，否則 `Y=15` |
| 2 | `1A38h..1A4Fh` | `X < 15` 時 `X++`，否則 `X=0` |
| 4 | `1A51h..1A68h` | `Y < 15` 時 `Y++`，否則 `Y=0` |
| 6 | `1A6Ah..1A81h` | `X > 0` 時 `X--`，否則 `X=15` |

位置欄位是 `DS:6A0Bh=X`、`DS:6A0Ch=Y`。更新後：

- `1A81h..1A92h` 以 `(X,Y)` 呼叫 resident `0131:003Eh`，結果寫 `DS:6A0Fh`；
- `1A92h..1AA2h` 以 `(X,Y,Facing)` 呼叫 resident `0131:0034h`，結果寫 `DS:6A0Eh`；
- `1AA8h` `retf`。

因此「這組欄位的 16×16 cardinal wrap 與更新後 refresh」是 `exact`，而不是從 CoAB
engine 猜來；「這組欄位就是一般地城隊伍位置」仍需 consumer／runtime anchor。

## 不能越級的結論

此函式在改寫座標前**沒有讀取牆 plane、門狀態或通行結果，也沒有失敗回滾路徑**。
所以它是「已獲准後的座標 commit wrapper」，不是完整 `CanMove`。只因它 wrap 就把
remake 接成 `geometry.CanMoveWrapped` 或 `CanMoveDungeonWrapped`，會漏掉 upstream
collision gate。

勘誤（2026-09-01）：overlay-07 的 **entry 27** 是 routine ordinal；ECL **opcode `27h`**
則是獨立的 `TREASURE` handler（見 Spec 032）。兩者數字相同不表示同一條呼叫鏈。
先前「從 opcode `27h` handler 回追移動」的下一步敘述錯誤，應改由 operand `1F40h`
在 TREASURE 固定物品載入路徑中的 consumer，以及 overlay-07 entry 27 的真實 callers
各自追查；不得再把兩者合併。後續移動切片需閉合：

1. 玩家 forward／turn input 如何映射；
2. current／target cell 哪一個 plane 與 direction bits 被檢查；
3. door／locked door 是否另有旗標或 mutation；
4. entry 27 是否真在一般移動 gate 成功後執行，以及失敗時的玩家可見結果。

在這四項完成前，remake 只可保留 Spec 011 的 scripted tour；不得開放自由移動。
