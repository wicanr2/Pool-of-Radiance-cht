# Spec 012：DOS cardinal coordinate wrapper 與 401Fh dispatch

狀態：READY（cardinal 座標更新、16×16 wrap、dispatcher call chain）；DRAFT（401Fh producer、牆／門 collision gate、玩家輸入 mapping）
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

## 401Fh dispatcher（READY）

overlay-03 entry 53（seed `3026h`）先由 resident command resolver 取得 dispatch value。
`30FAh` 原始 bytes `3D 1F 40` 比對 `401Fh`；相等時 `30FFh` 原始 bytes
`9A A7 00 45 00` 呼叫 overlay-07 entry 27 的 resident stub。另一個相鄰值 `4019h`
在 `3106h` 走不同分支，只更新 view detail，不得和 movement wrapper 合併。

這證明 `401Fh → overlay-07 entry 27` 的 dispatch chain，推論等級為 `exact`；目前尚未
閉合「哪一個玩家輸入／牆判斷 producer 會產生 401Fh」。

## overlay-07 entry 27（READY）

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

因此 16×16 cardinal wrap 與 post-move refresh 是 `exact`，而不是從 CoAB engine 猜來。

## 不能越級的結論

此函式在改寫座標前**沒有讀取牆 plane、門狀態或通行結果，也沒有失敗回滾路徑**。
所以它是「已獲准後的座標 commit wrapper」，不是完整 `CanMove`。只因它 wrap 就把
remake 接成 `geometry.CanMoveWrapped` 或 `CanMoveDungeonWrapped`，會漏掉 upstream
collision gate。

下一個 READY 切片必須從 overlay-03 entry 53 使用的 resident command resolver 往
`401Fh` producer 回追，閉合：

1. 玩家 forward／turn input 如何映射；
2. current／target cell 哪一個 plane 與 direction bits 被檢查；
3. door／locked door 是否另有旗標或 mutation；
4. gate 成功才送出 `401Fh`，失敗時的玩家可見結果。

在這四項完成前，remake 只可保留 Spec 011 的 scripted tour；不得開放自由移動。
