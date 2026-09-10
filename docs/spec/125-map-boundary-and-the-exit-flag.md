# Spec 125：走到地圖邊界會怎樣——`@6DD5` 是誰寫的

狀態：CONFORMED（`06AEh` 的完整行為、`[4937h]+5AAh` ＝ ECL `6DD5h` 的換算、
真正的前進在 `CALL C01Eh`、以及**地圖是 wrapped 不是 bounded**）。
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

## 真正的前進在 `CALL C01Eh`——而且是繞的

`06AEh` **只在越界的那四條路上寫 `ds:6A0Bh`／`6A0Ch`**，沒越界的那一步一個字
都沒寫。它的唯一呼叫端是 overlay-14 `0AF8h`——鍵盤分派裡掃描碼 `48h`（上鍵）
那一支——所以它確實是「按前進」的入口，不是別的東西。

**那越界時寫的那一下是什麼？是 no-op。** 新座標是從目前座標算出來的：
X 是 0 往西走得到 −1，夾回 0——本來就是 0；X 是 15 往東走得到 16，夾回 15
——本來也是 15。**四條路的寫入值一律等於原值**，所以那不是「把隊伍拉回邊界」，
只是與 `@6DD5 = 1` 成對的一個動作。真正的訊號是旗標。

接著那一格的 ECL 跑起來。它若不換圖，就用 `2Dh CALL C01Eh` 提交這一步——
選擇子 `C01Eh` 是 **overlay-07 entry 27（`1A17h`）**，內容逐朝向寫死：

```
朝向 0（北）  ds:6A0Ch > 0  ? --  : = 0Fh
朝向 2（東）  ds:6A0Bh < 0Fh? ++  : = 0
朝向 4（南）  ds:6A0Ch < 0Fh? ++  : = 0
朝向 6（西）  ds:6A0Bh > 0  ? --  : = 0Fh
```

**這就是 16×16 的繞回。** 所以答案是 **wrapped**：走到邊緣再走一步會出現在
對邊，除非那一格的腳本先用 `@6DD5` 把你帶去別張圖。

`1A89h` 之後還把 `0131:003Eh`（overlay-30 entry 6）的結果存進 `ds:6A0Fh`
——那是新位置的牆位元組（spec 015 的 `C04F` 那一組）。

## 還沒讀：overlay-03 也碰 `[4937h] + 5AAh`

`cmd/pool-disp-scan -addresses 5AA` 掃出六處，其中五處在 overlay-14
（`06C1h` 一處、`0700h`／`0716h`／`072Ch`／`0742h` 就是上面那四條寫 1 的），
**第六處在 overlay-03 `362Dh`**，形狀同樣是 `89 85 AA 05`
（`mov [di+5AAh], ax`）。

本節整支只讀了 overlay-14，所以 overlay-03 那一處在做什麼、它的 `di` 是不是
也指向 `[4937h]`，都還沒確認。**在確認之前不要把「只有 overlay-14 寫這個
旗標」當成定論**——那是這一份的掃描面決定的，不是原版決定的。

## 對 remake 的意思

**現況是對的，不要改。** `moveInitialDungeonForward` 用 `WrapCoordinate`
與 `C01Eh` 逐條相同；`CanMoveDungeonWrapped` 的繞回與 `0131h:0039h` 相同。

曾經照 `06AEh` 的夾去改過一次，結果 `TestAWildernessStepMovesBothPositions`
與 `TestTheWildernessWalkReachesTheWesternSheet` 都紅——**那不是野外的特例，
是把前置當成了提交**。`06AEh` 的夾是 no-op，提交在 `C01Eh`，而提交是繞的。
