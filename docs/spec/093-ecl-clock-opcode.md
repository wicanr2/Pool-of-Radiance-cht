# Spec 093：`34h ECL CLOCK`

狀態：READY（運算元個數、推進的目標與次數已讀出並實作）；DRAFT（時鐘七格
的語意、進位常式、以及原版讀運算元編號的方式）。
日期：2026-09-04（原 2026-09-03）。

## 它做的事

overlay-03 `2E1Ah` 取一個運算元，把讀到的值當作次數傳給
overlay-20 entry 2（`0392h`），單位固定是 **1**（`2E38h` 推的常數）。

`0392h(unit, count)`：

```
0398h  從 [4933h] + 6E00h 起把七個 word 搬進區域變數
03CCh  重複 count 次：clock[unit]++，然後呼叫 02B1h（進位）
```

所以時鐘是**七個欄位**，推進的方式是「加一再進位」重複 count 次，
不是直接加 count。

## 它吃**一個**運算元，不是兩個（2026-09-04 訂正）

`2E1Ah` 的序言是 `mov al, 1 / push ax / lcall 0045h:002Ah`——取**一個**
運算元。共用 engine 的 `ecl.KnownCommands` 寫兩個，那張表是二手的
（公開的 ECL dump 表加上 CoAB 的重製），而 **CoAB 的 `34h` 確實是兩個**
（`ECL CLOCK 4C06 04`）：運算元個數是各作品直譯器 build 的性質，
不是格式的性質。

差這一個運算元的後果不是報錯，是**整段錯位**。全遊戲唯一的呼叫點
`ECL7/block 17 @9D37h` 之後接的是 `GOSUB 9DA7h`：

```
9D37  34 00 01           ECL CLOCK 1
9D3A  02 01 A7 9D        GOSUB 9DA7h
9D3E  03 01 82 6E 00 FF  COMPARE @6E82, 255
```

讀成兩個運算元的話，`02 01 A7` 被吃成第二個運算元（先前的規格把它記成
「字組字面值 `A701h`」），PC 停在 `9D3Dh`——那裡是 `9Dh`，不是 opcode。
靜態走訪因此在這裡停住，`ECL7/block 17` 三個入口有兩個追不完；
VM 走到這裡也一樣會停。

Pool 這一側的指令表由 `gamepack.PoolCommandTable()` 提供：底稿是共用 engine
那張，**只有 `34h` 這一條覆蓋**。`TestPoolCommandTableMatchesTheMeasuredChain`
拿 overlay-03 真檔量一次比對，不一致的清單多一條就會紅。

## 一個原版的怪處

`2E28h` 讀的是 `[bp-1]`——一個**沒有初始化過的堆疊位元組**——再拿它當
「要讀第幾個運算元」。既然只取了一個運算元，那個索引指到別處時讀到什麼
還沒讀出來；唯一的呼叫點傳的是位元組字面值 `1`，remake 就讀運算元 1。

## remake 現況

時鐘存在 app 裡（七個 word），`34h` 對第 1 格加 count 次一。
**進位沒做**：`02B1h` 還沒讀，而時鐘目前沒有任何取用點，
所以進位錯了也看不出來、影響不到玩家路徑。接上取用點之前不要靠它。

## 指令表要一路帶到「下一條在哪」那一步

`34h` 是 **passthrough**（引擎沒有核心處理常式，由前端接），而 VM 的
passthrough 分支原本用 `ecl.RecordEnd` 算下一條指令的位址——那一支沒有指令表
參數，一律用共用 engine 那張二手的。於是同一條指令出現兩套長度：**內容用
Pool 的表解（一個運算元）、位址用二手表算（兩個運算元）**，PC 多跳三個位元組。

症狀延後發作，這是它難查的地方：這一條照樣執行完、事件照樣送出去，PC 卻停在
指令中間，要等到走到下一條才冒出 `unknown opcode`。實際撞到的是
`ECL7/block 17` 的 `9D37h`（payload offset 1079）：

```
payload 1075..1090   01 FF 9C 13 34 00 01 02 01 A7 9D 03 01 82 6E 00
                                 ^1079 34h
Pool 的表    1079 → 下一條在 1082（02h GOSUB）
二手表       1079 → 下一條在 1085（9Dh，不是 opcode）
```

engine 那一側現在是 `ecl.RecordEndWithCommands`，VM 傳自己的 `commands`
（`TestPassthroughAdvancesWithTheSessionCommandTable`）。Pool 這一側每一個建
session 的入口都要 `SetCommands(PoolCommandTable())`——`NewDOSECLArchiveSession`
（讀檔那條路）原本漏了。

**可重用的規則：解碼用的表要跟著走完整條路徑。** 只要有一個地方退回預設，
出錯的就不是那一條指令，而是**它之後的每一條**——而報出來的位置離成因很遠。
