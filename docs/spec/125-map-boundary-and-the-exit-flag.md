# Spec 125：走到地圖邊界會怎樣——`@6DD5` 是誰寫的

狀態：READY（`06AEh` 的完整行為、`[4937h]+5AAh` ＝ ECL `6DD5h` 的換算、
查詢面繞回與位置面不繞的分工）；
DRAFT（沒越界的那一步是誰前進的、因此 bounded／wrapped 尚未定案）。
日期：2026-09-06。

## 這一份解掉 spec 100 的 OPEN

spec 100 的狀態寫著「OPEN（誰寫它、寫的是什麼值）」——29 個 ECL 區塊裡有 18 個
在入口 0 讀 `DS:6DD5h`，卻找不到任何 ECL 寫它。

**寫它的是 overlay-14 `06AEh`**，而且不是用 ECL 位址寫的，是引擎直接寫記憶體：

```
06AE  f()
06B4  X = ds:6A0Bh、Y = ds:6A0Ch、朝向 = ds:6A0Dh      ; spec 106 的 class 4
06CA  [4937h] + 5AAh = 0                              ; 先清掉
06E1  al = 0131:0039(X, Y, 朝向)                       ; 門／牆的狀態（spec 122）
06E6  al == 0 → return                                ; 過不去就整支不做
06EA  X += DS:274Ah[朝向]                              ; 九方向位移表
06F5  Y += DS:2753h[朝向]
0700  X > 15 → ds:6A0Bh = 15、[4937h]+5AAh = 1
0716  X < 0  → ds:6A0Bh = 0 、[4937h]+5AAh = 1
072C  Y > 15 → ds:6A0Ch = 15、[4937h]+5AAh = 1
0742  Y < 0  → ds:6A0Ch = 0 、[4937h]+5AAh = 1
0758  return
```

### `[4937h] + 5AAh` 就是 ECL 的 `6DD5h`

spec 106 的 class 1（`6B00h..6EFFh`）取法是 `[4937h] + 2A00h + addr × 2`。
反解：`2A00h + addr × 2 ≡ 5AAh (mod 10000h)` → `addr × 2 = DBAAh` → `addr = 6DD5h`。

所以「走到邊界」與「地圖出口閘門」是同一件事的兩端：
**引擎立旗標，腳本讀旗標決定要不要換圖**。spec 100 說「remake 從來沒有寫過它，
所以兩條路都是死的」——現在知道原版是誰寫、什麼時候寫、寫什麼值。

## 查詢面繞回，位置面不繞

`0131h:0039h`（overlay-30 entry 5，`0358h`）**自己把 X／Y 繞回 0..15**
才去查那一格的牆：

```
0383  [bp+0Ah] > 0Fh → 0        ; Y
0397  [bp+0Ah] <  0  → 0Fh
0397  [bp+08h] 同樣處理          ; X
```

同一個慣用法在 overlay-30 出現六次，其他 overlay 一次都沒有。
**所以「牆的判定繞回去」是原版行為**，remake 的 `CanMoveDungeonWrapped`
對得上；而位置在 `06AEh` 是被夾住的。**查詢繞、位置夾——兩件事，不要混。**

## 為什麼 bounded／wrapped 還沒定案

`06AEh` **只在越界的那四條路上寫 `ds:6A0Bh`／`6A0Ch`**；沒越界的那一步
它一個字都沒寫。所以真正的前進發生在還沒讀到的呼叫端，
不能從這一支斷定「位置一律夾」。

實測也擋著：把 remake 改成夾之後，`TestAWildernessStepMovesBothPositions`
與 `TestTheWildernessWalkReachesTheWesternSheet` 都紅——**野外那三張圖的
16×16 貼圖（spec 105）靠的就是繞回去**。

兩種讀法都還活著：

1. 城區／地城夾、野外另有一條路；
2. `06AEh` 是「這一步會不會出圖」的前置，真正的前進在呼叫端而且會繞。

**在呼叫端讀出來之前不動 remake 的行為。** 現況（位置繞、牆的判定也繞）
與野外的實測相符，改成夾會弄壞已經驗過的東西。
