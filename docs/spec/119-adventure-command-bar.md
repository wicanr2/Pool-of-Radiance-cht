# Spec 119：冒險畫面的指令列與平面全圖

狀態：READY（兩條指令列的字串、六個鍵的分派、`A)REA` 的開關與平面圖畫法、
`S)EARCH`／`L)OOK` 的旗標位、`C)AST` 那一支的流程與它呼叫的兩顆 overlay）；
DRAFT（`+594h` 兩個旗標在搜尋裡改變什麼、`V)IEW` 那一頁的版面）。
日期：2026-09-08。

## 兩條字串（exact）

`START.EXE` 的 DS 區有兩條 Pascal 字串：

| 位址 | 長度 | 內容 |
|---|---:|---|
| `DS:04CAh` | `21h` | `Area Cast View Encamp Search Look` |
| `DS:04F3h` | `1Ch` | `Cast View Encamp Search Look` |
| `DS:051Ch` | `1Fh` | `Save View Magic Rest Alter Exit`（紮營主選單）|

**第二條少了 `Area`**。說明書 p.21 寫「在月之海沿岸的陸地上行動時，此指令
完全沒有作用」——那不是「按了沒反應」，是**野外那一列根本沒有這一項**。

## 分派（exact，overlay-14 `09CAh`）

那一支先呼叫 `011Dh:002Fh`（overlay-26）把字串畫成一列並收一個鍵，
回傳的字母在 AL：

| 鍵 | 做什麼 |
|---|---|
| `A` `41h` | `[4933h]+1F6h == 0` 才有效：把 `DS:6A0Ah` 在 0／1 之間翻轉，再呼叫 `0131h:0048h`（overlay-30）帶三個座標 `6A0Bh/6A0Ch/6A0Dh` 重畫。不為 0 就印另一句話——說明書的「有些區域無法顯示平面全圖」。|
| `C` `43h` | 選定的角色 `[5CF0h]+10Ch == 0`（狀態正常）才呼叫同 overlay 的 `0AEAh`。|
| `V` `56h` | 呼叫 `00C9h:0039h`（overlay-19 entry 3）。|
| `E` `45h` | 設區域變數 `var_2 = 1`，迴圈結束後進紮營。|
| `S` `53h` | `[4937h]+594h ^= 1`。|
| `L` `4Ch` | `[4937h]+594h |= 2`，呼叫 `00D2h:002Ah`（overlay-20 entry 2）帶 `(2,1)`，再把 ECL 的 PC 設成 `ds:4946h`——那是**五個入口位址的第二個**，也就是 spec 016 的 `SearchLocation`（entry 1）。|

所以 `+594h` 的第 0 位是「邊走邊搜」、第 1 位是「這一次要搜」。

## `C)AST` 走到哪裡（exact，2026-09-08）

`0A84h` 那個 `call far` 的位元組是 `9A 2A 00 AC 00`，也就是 **`00ACh:002Ah`**。
查 stub 對照表（spec 109）與 TPOV 清冊：`00ACh` 是 overlay-15，
stub 位移 `2Ah` 是它的 **entry 2**，code offset `0497h`。

`0497h` 的流程：

```
049D  var_4 = 0                       ; 收尾要不要重畫
04A1  var_2 = FFFFh                   ; 記憶槽位，FFFFh ＝ 還沒選
04A6  ds:6CA9h = ds:6CABh = 0
04AE  sub_35D(1)                      ; 前置檢查，回 0 就直接收尾
04BC  迴圈頂：
      00C9h:005Ch(&var_3, &var_2, 1, 0)   ; overlay-19 entry 12（0297Bh）
04D1  var_6 = al                          ; 選中的法術編號，0 ＝ 沒選
04D4  var_6 ≠ 0：
        var_4 = 1
        sub_1500(0, 16h, 26h, 11h, 1)     ; 開文字框
        sub_1638(16h, 26h, 11h, 1)
        00E2h:0039h(&var_5, 1, 0, var_6)  ; overlay-22 entry 5（0C14h）
        var_2 = FFFFh                     ; 施完把槽位清掉，回迴圈頂
051E  var_3 ≠ 0：var_4 = 1，離開迴圈       ; 玩家按了取消
052A  兩者都是 0：把選定角色（ds:5CF0h／5CF2h）的名字接上 `047Fh` 的
      `has no spells memorized` 印出來
054C  var_6 ≠ 0 就跳回迴圈頂——**施完一條可以接著施下一條**
0555  var_4 ≠ 0 → sub_1179()             ; 重畫冒險畫面
```

三件事因此定了下來：

1. **探索時的施法與戰鬥中是同一支**：`00E2h:0039h` ＝ overlay-22 entry 5，
   正是 spec 073 那張 67 格派發表的入口（`0C14h`）。所以效果不必另外解，
   remake 這一側也應該共用 `cast.go` 那條路。
2. **挑人與挑法術是一個介面**：overlay-19 entry 12 一次回傳「取消旗標、
   記憶槽位、法術編號」三樣。
3. **選到沒記法術的人會說 `<名字> has no spells memorized`**（`047Fh`，
   Pascal 長度 23）。那不是錯誤訊息，是正常回應。
`24h COMBAT` 的處理常式（overlay-03 `186Ch` 內的 `19A1h`）也會寫這個欄位。

## 平面全圖

說明書 p.21：「圖上僅顯示牆而不顯示門」，隊伍「以一個箭號表示，箭頭所指的
方向就是隊伍前進的方向」。remake 照這條畫：16×16 的格線，每一格照 GEO 的
四個方向牆位元組決定畫不畫那一條邊，隊伍那一格畫一個指著朝向的箭頭。

## 契約

1. 指令列只在**自由移動**時出現。導覽還在跑、對話框開著、戰術地圖上都不畫
   ——原版那時最下面是 `PRESS <ENTER>/<RETURN> TO CONTINUE`（基準圖 `03`），
   自由移動之後才換成指令列（基準圖 `06`）。
2. 野外那一列不含 `AREA`，`A` 也不接受。
3. `S` 翻 `SearchWhileWalkingBit`，狀態列跟著多一段 `SEARCH`
   （說明書 p.20 的狀態列範例是 `15,4 N 12:33 SEARCH`）。
4. `L` 設 `LookOnceBit` 並對現在這一格重跑 ECL entry 1，跑完清掉那一位。
5. `A` 切換平面全圖；開著的時候那一框畫平面圖而不是第一人稱視野。
6. F-key 提示移到 F1 說明頁，不是拿掉。

## 驗收

- `TestAdventureCommandListDropsAreaInTheWilderness`：兩條列的內容。
- `TestSearchToggleShowsInTheStatusLine`：`S` 一開一關，狀態列跟著變。
- 截圖 `docs/screenshots/pool-remake-free-movement.png` 與
  `pool-remake-area-map.png`（都由 `tools/capture-remake-creation.sh` 重生）。

## 仍未閉合

- **`+594h` 那兩位在搜尋裡改變什麼還沒讀**。remake 目前只做得出可見的部分：
  狀態列的字樣、`L` 重跑一次 entry 1。真正的差別（搜得到什麼）要等讀出
  消費端。
- `C)AST` 與 `V)IEW` 現在開的是 remake 自己的法術清單與裝備頁，不是原版的
  施法流程與角色資料頁（`ITEMS SPELLS DROP EXIT`）。不宣稱與原版一致。
- **門的第一人稱美術沒有畫**：`(0,4)` 朝西正前方是城門，
  `TraverseWallViewWrapped` 對它送出 `wallType=1`，而 `BuildWallLayout`
  一片圖章都產不出來，對拍因此只有 60.4%
  （`TestFirstPersonInsetMatchesTheDOSShot` 已釘住）。

  **原因查到一半**：WALLDEF record 0 slice 0 **不是空的**，layout 6 那一段
  （索引 54..109）長這樣——

  ```
  01 01 01 01 01 01 01   28 0D 0D 0D 0D 0D 27
  28 0E 0E 0E 0E 0E 27   28 0E 0E 0E 0E 0E 27
  28 0E 0E 0E 0E 0E 27   28 0E 0E 7C 0E 0E 27
  28 0E 22 03 21 0E 27   28 22 03 03 03 21 27
  ```

  形狀就是一道有拱的門面。掉光的原因是 `WallSymbolItem` 用
  `item = id − SymbolSetBase[記錄的符號集]`（record 0 是 `0x2E`）算索引，
  而這些編號**低於 `0x2E`**，一律算成負的就被丟掉。共用 engine 自己的註解
  說得很清楚：原版的 `PUT8X8SYMBOL` 是**看編號自己落在哪一帶**決定去哪一組
  取圖，`0x01`／`0x2E`／`0x74`／`0xBA` 是四帶的起點，而第 0 帶（`0x01` 起）
  是「另一組 8X8D」。

  **兩個直覺的假說都試過，兩個都不對**（拿兩張基準圖各量一次）：

  | 假說 | `(0,4)` | `(14,1)` |
  |---|---:|---:|
  | 現況（低號帶直接丟掉）| 60.4% | **96.1%** |
  | 低號帶 → `8X8D3` 區塊 1，`item = id − 1` | 73.3% | 76.7% |
  | 低號帶 → `8X8D3` 區塊 3 | 62.1% | 72.1% |
  | 低號帶 → `8X8D3` 區塊 5 | 74.2% | 73.3% |
  | 低號帶 → 那一筆 record 自己的區塊，`item = id − 1` | 64.8% | 71.6% |

  每一種都讓 `(0,4)` 變好、卻讓 `(14,1)` 從 96% 掉到七十幾——**所以第 0 帶
  不是這三塊裡的任何一塊，也不是 record 自己那塊**。下一步是讀
  `PUT8X8SYMBOL`（原版單元 `SQRPAK8`）的實際帶對應，不要再猜。
