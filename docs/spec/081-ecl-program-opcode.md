# Spec 081：`38h PROGRAM`

狀態：READY（值 0 與值 9 都已接上，兩者都有按鍵測試）；DRAFT（值 9 的休息
時間與被打斷那一段、隊伍管理畫面本身的選單項目、值 8 的圖）。
日期：2026-09-04（原 2026-09-03）。

## 為什麼卡在這裡

只用按鍵在起始地圖上走，走進某一格會停在
`opcode 0x38 at 1865 has no core handler or adapter passthrough`。
那一格是 ECL3／block 11 的 `PROGRAM 0`。

## 四個呼叫點，三個值

以 `ecl.TraceGraphAtBase`（只走得到的碼）掃八個封存檔會得到三條 `38h`；
再用**線性指令掃描**（`ecl.ScanKnownInstructions`）掃一次會多出第四條，
在 `ECL5`／block 7。差別的原因很單純：那個 block 是三個靜態展開解不開的
之一，走得到的碼掃不進去。

| 位置 | 運算元 | 在做什麼 |
|---|---:|---|
| `ECL3`／block 0 `A1ADh` | 9 | 城裡問一句再開隊伍管理 |
| `ECL3`／block 11 `A049h` | 0 | 直接開隊伍管理 |
| `ECL7`／block 17 `A5EFh` | 9 | 同 block 0 |
| `ECL5`／block 7 `A82Ah` | **8** | **打贏泰倫斯拉克斯之後的結局過場** |

第四條的上下文逐條讀得出來（線性掃描）：

```
a802  COMBAT                       ; 最後一戰，LOAD MONSTER 42h ＝ mon5/66
a80e  COMPARE @4ABA, FFh ; IF <>
a815  SAVE FEh → @4ABA             ; 破關旗標（市政廳槽 20）
a81b  OR @4A6D, 10h → @4A72
a824  SAVE 01 → @4AE0
a82a  PROGRAM 08                   ; ← 結局過場
a82d  PRINTCLEAR "KNOWING THAT TYRANTHRAXUS HAS FINALLY BEEN DEFEATED, ..."
a8e5  座標設回 (0,4) 朝向 1、6E12 = 3、NEWECL 0
```

## 派發（overlay-03 `3167h`）

```
316Dh  [4393h] 非 0 時 [5CF0h..5CF2h] = [4394h] 的 far pointer；[4393h] = 0
3186h  取運算元 1（overlay-07 entry 2）再讀值（overlay-07 entry 1）
319Eh  值 = 0 → overlay-16 entry 1（code 014Eh）
                再 overlay-25 entry 37（code 280Fh）
31AFh  值 = 8 → overlay-18 entry 1（code 02A1h）
31BBh  值 = 9 → 存下 ECL PC（`494Eh`）→ 近呼叫 312Ah → 還原 PC
                → 近呼叫 0001h（`00h EXIT` 的 handler）
```

其餘的值直接返回，什麼都不做。

`494Eh` 存了又還原這件事說明 `312Ah` 底下那條鏈**會動到 ECL PC**
（它會跑別的 ECL），所以回來之後要接回原來的位置，然後才讓 block 結束。

## 值 9：旅店的過夜

```
312Ah  flag = 0
3134h  近呼叫 35DBh，參數是 [4948h]
313Ch  overlay-15 entry 1（code 1E45h），輸出一個 byte 到 flag
3146h  flag 非 0 時：overlay-25 entry 37（code 280Fh）
                    近呼叫 35DBh，參數是 [494Ah]
3159h  overlay-27 entry 1（code 0181h）
315Eh  [54DBh] = 0
```

**全遊戲只有一處推值 9**：`ecl3/0` 的 `A1ADh`，就在旅店那一段裡——

```
A140  COMPARE @4A07, 0 ; IF <> ; EXIT
A14E  PICTURE 24
A15C  PRINTCLEAR "'IT WILL COST YOU 1 PLATINUM PIECE TO REST HERE.  DO YOU WANT TO STAY?'"
A195  GOSUB AE5A               ; YES／NO
A199  ON GOTO [A1A5, AA38]
A1A5  GOSUB A1B4               ; WHO 'WHO WILL PAY?' → 扣一枚白金
A1A9  GOSUB 9A63               ; 設 6DD2／6DD3
A1AD  PROGRAM 9
A1B0  PICTURE 255
A1B3  EXIT
```

（城區地形索引 9 那七格，spec 102。）派發鏈的**第一支是 overlay-15 entry 1**，
而 overlay-15 整支都在動記憶法術陣列（spec 070，23 個進入點）——所以值 9 是
**「選法術＋休息」**，不是隊伍管理。回傳非 0 才接 overlay-25 entry 37。

## 值 8：結局過場（overlay-18 entry 1，`02A1h`）

`02A1h` 用 `DS:52D4h` 組出檔名放進 `DS:4675h`、把 `DS:4954h` 存起來再設成 7、
清掉 `4670h`／`4672h`，接著用 `0198h:0066h`／`0198h:002Fh` 一段段畫框與文字。
它畫的是**結局**——字串就內嵌在同一顆 overlay 的 `0111h..02A0h`：

```
0111  "Mortally wounded, the dragon roars!"
0135  "The spirit of Tyranthraxus flares up"
015A  "from the dragon's body."
0172  "FINAL"
0178  "\"Fools, you have but slain the body"
019C  "I possessed.  I cannot be defeated!\""
01C1  "With the power of the pool of radiance"
01E8  "which I moved here, to my lair,"
0208  "I will still rule, by possessing one"
022D  "of you!\""
0236  "\"No Lord Bane!  I can still rule here!"
025D  "I have not failed.  Do not call me back"
0285  "through the pool!\""
0298  "Noooo..."
```

**remake 目前把值 8 當成什麼都不做**，所以打贏之後直接跳到 `A82Dh` 的
文字。那是已知的缺口，不是原版行為。

## 值 0：直接開隊伍管理

overlay-16 entry 1（`014Eh`）把 `52D4h` 設為 3（spec 006 認得這個值：
portrait renderer 的模式），清掉 `466Eh` 與 `4954h`，然後用
`0634h`／`0310h` 那一對畫視窗與跑選單迴圈。overlay-25 entry 37（`280Fh`）
依 `4954h` 分支，是同一個畫面的主迴圈。

overlay-16 是建角那個 overlay（spec 003／006／072 都引用它），所以這兩支
合起來就是**城裡的隊伍管理／訓練所畫面**。

## 還沒讀

- `35DBh`（overlay-03 內）拿 `4948h`／`494Ah` 當參數做什麼。
- overlay-16 entry 1 與 overlay-25 entry 37 的選單項目與各自的行為。
- overlay-18 entry 1 的畫面流程（誰按什麼往下翻、`FINAL` 那張圖從哪裡來）。
- `4393h`／`4394h` 那個 far pointer 由誰寫。

## remake 現況

值 0 已接上：`38h` 進 passthrough，前端讀運算元 1，值 0 就把畫面切到隊伍管理
（`programManaging`），B 或 ESC 回地圖並讓 ECL 從原地繼續。**從地圖進來的那次
不能走「開始冒險」那條路**——那會把開場整個重跑，隊伍被丟回起點，而畫面上
看起來只是「怎麼又在講故事」。

值 9 接的是**紮營畫面**（選法術與休息，見 `camp.go`）。ECL 自己已經問過
「要不要花一枚白金住下」也收過錢了，所以前端不再多問一次；收掉畫面之後
**這個 block 就結束**——原版在那一支的結尾呼叫的正是 `00h EXIT` 的 handler，
所以 remake 走的是同一條收尾路徑（`finishCellBlock`）。

> 先前這裡接的是「問一句再開隊伍管理」，玩家付了一枚白金什麼都沒得到。
> 推翻它的是兩件事：值 9 全遊戲只出現在旅店，而它的派發鏈第一支是
> overlay-15（記憶法術）。

`TestTheInnChargesAPlatinumAndOpensCamp` 走完整段：走進旅店、答應、
白金少一枚、紮營開起來、休息把整隊治滿、收掉之後回到地圖。

**時間還沒接**：原版的紮營讓玩家挑天／時／分，也會被打斷（`Stop Resting?`），
`9A63h` 寫的 `6DD2`／`6DD3` 看起來就是那一段的參數，還沒讀。

其餘的值照原版直接繼續，什麼都不做。

驗收：`TestNormalKeysReachThePartyManagementCell` 只用按鍵走到 ECL3／block 11
那一格，開起畫面、按 B 回地圖，斷言位置沒變、開場沒有重跑、而且還走得動。
