# Spec 073：法術效果的派發表

狀態：CONFORMED（表的位址與版面、67 格的內容、每一格指到哪一支 overlay-22
常式、六組共用；由 `internal/gamepack` 靜態解出並以四條測試鎖住）；
DRAFT（57..67 這 11 個無名編號的來源、各支處理常式的內容）。日期：2026-09-03。

## 一次呼叫就分岔到 67 支常式

overlay-22 的 entry 5（code `0C14h`）在 `0E92h` 這一段把法術編號寫進
`DS:6779h`，再用它當索引作一次遠間接呼叫：

```
0E92  8A 46 0E        mov  al, [bp+0Eh]        ; 法術編號，1 起算
0E95  A2 79 67        mov  ds:6779h, al
0E98  A0 79 67        mov  al, ds:6779h
0E9B  30 E4           xor  ah, ah
0E9D  8B F8           mov  di, ax
0E9F  D1 E7           shl  di, 1
0EA1  D1 E7           shl  di, 1               ; di = id × 4
0EA3  FF 9D 78 6A     call dword ptr [di+6A78h]
0EA7  C6 06 79 67 00  mov  byte ptr ds:6779h, 0
0EAC  C6 06 7E 67 00  mov  byte ptr ds:677Eh, 0
```

`DS:6779h` 在呼叫前寫、呼叫後清成 0，所以它是「現在正在處理哪一個法術」，
不是持久狀態；處理常式讀得到它。

## 表在 BSS，但對應關係是靜態的

`6A78h + 30640 = 57880`，已經超過 START.EXE 的 47936 bytes，所以表不在檔案裡，
執行期才填。填表的是 overlay-22 的 entry 10（code `33CCh`）：67 段一模一樣的

```
B8 lo hi        mov ax, 處理常式的 stub 位移
BA lo hi        mov dx, 00E2h                  ; overlay-22 自己的控制段
A3 lo hi        mov ds:槽,   ax
89 16 lo hi     mov ds:槽+2, dx
```

一段 13 bytes，位置從 `33F3h` 排到 `374Dh`。**所以「法術編號 → 處理常式」
不必跑遊戲就能完整解出來**，`internal/gamepack.ParseSpellDispatchTable`
就是在做這件事，結果 dump 在
[`docs/audit/pool-spell-dispatch.json`](../audit/pool-spell-dispatch.json)。

解出來的東西要通過兩道檢查才算數：遠指標的**段號必須等於這個 overlay 自己的
控制段**（只比位移會把巧合收進來），而且 stub 位移要能在進入點表裡查到。

| 名稱 | 位址 | 內容 |
|---|---|---|
| 表基底（法術 1）| `DS:6A7Ch` | 67 個遠指標，編號 1..67 |
| 目前法術編號 | `DS:6779h` | 派發前寫入、派發後清零 |
| 處理常式 | overlay-22 | 53 支不同的常式 |

## `6A78h` 不是表的第 0 格

表看起來從 `6A78h` 開始——`[di+6A78h]` 就是這樣寫的——但那一格是另一個東西：

- `0D65h` 與 `30FAh` 兩處用 `call dword ptr ds:6A78h` **直接**呼叫它，而且會先
  推四個引數（`30F8h` 那處推兩個 byte 加一個遠指標）。派發點那個間接呼叫
  **一個引數都不推**。兩種簽章不可能指向同一支常式。
- overlay-08 的 `0060h` 與 `007Bh` 會把它改寫成別的值（`00E2h:0034h`、
  `0096h:007Ah`）。程序變數會被改寫，法術處理常式不會。

所以 `6A78h` 是一個獨立的程序變數，表從下一個 double word 開始。這剛好讓
`di = id × 4` 對上 1 起算的編號——`id = 1` 落在 `6A7Ch`。
編號 0 沒有意義，走到那一格等於呼叫那個程序變數。

## 共用處理常式的六組，就是這張表的語意證據

53 支常式服務 67 個編號，多出來的 14 格全部落在語意上重複的法術上：

| 處理常式 | 編號 | 法術 |
|---|---|---|
| `10D1h` | 5, 11, 18, 22, 29 | Detect Magic ×2、Read Magic、Find Traps、Detect Invisibility |
| `110Bh` | 6, 7, 16, 17, 52, 53, 54 | 七個 Protection From … |
| `1650h` | 23, 49 | Hold Person（牧師／法師各一份）|
| `19FBh` | 30, 50 | Invisibility、Invisibility, 10' Radius |
| `2356h` | 41, 46 | Dispel Magic（牧師／法師各一份）|
| `262Eh` | 47, 64 | Fireball，以及編號 64 的無名效果 |

名稱表（spec 068）裡有五組**完全同名**的法術：Detect Magic、Protection From
Evil、Protection from Good、Hold Person、Dispel Magic。五組全部落在同一支常式
上，沒有例外。這是可以被推翻的預測——只要有一對分到不同常式，「依法術編號
派發」的讀法就垮了——它通過了，所以編號的意義是**exact**，不是猜的。

`10D1h` 那一組是反面補充：五個名稱不同的法術共用一支，共通點是它們在戰術地圖
上都不產生效果。**推論方向只有一個**：同名必共用（已驗證）；共用不必同名。

## 57..67 這 11 個編號沒有名字

名稱表只有 56 筆，表卻有 67 格。編號 64 與 Fireball 共用常式，指向「物品與
怪物的特殊效果借用同一條派發路徑」（wand、necklace 這類）。**這是待證**：
還沒有找到寫入 57..67 的呼叫端。

## 尚未解讀

- 53 支處理常式各自做什麼（spec 068 目前掛的是手冊敘述，不是反組譯結論）。
- `DS:677Eh` 為什麼在派發之後跟著清零。
- 呼叫 entry 5 的上游怎麼決定 `[bp+0Eh]`，以及 57..67 從哪裡進來。
