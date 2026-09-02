# Spec 081：`38h PROGRAM`

狀態：READY（值 0 已接上並有按鍵測試）；DRAFT（值 9 的問句字串與
block 結束路徑、以及隊伍管理畫面本身的選單項目還沒讀）。日期：2026-09-03。

## 為什麼卡在這裡

只用按鍵在起始地圖上走，走進某一格會停在
`opcode 0x38 at 1865 has no core handler or adapter passthrough`。
那一格是 ECL3／block 11 的 `PROGRAM 0`。

## 全遊戲只有三個呼叫點

掃過八個 ECL 封存檔的每一個 block（`ecl.TraceGraphAtBase`，起始位址
`0x9900`）只有三條 `38h`，運算元都是位元組字面值：

| 位置 | 運算元 |
|---|---|
| `ECL3`／block 0 offset `08ADh` | 9 |
| `ECL3`／block 11 offset `0749h` | 0 |
| `ECL7`／block 17 offset `0CEFh` | 9 |

兩個值都落在原版真的有處理的分支上，所以這條 opcode 不需要通用實作，
只要這兩支。

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

## 值 9：問一句，答應就開隊伍管理

```
312Ah  flag = 0
3134h  近呼叫 35DBh，參數是 [4948h]
313Ch  overlay-15 entry 1（code 1E45h），輸出一個 byte 到 flag
3146h  flag 非 0 時：overlay-25 entry 37（code 280Fh）
                    近呼叫 35DBh，參數是 [494Ah]
3159h  overlay-27 entry 1（code 0181h）
315Eh  [54DBh] = 0
```

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
- 值 8 的 overlay-18 entry 1（全遊戲沒有呼叫點，先不讀）。
- `4393h`／`4394h` 那個 far pointer 由誰寫。

## remake 現況

值 0 已接上：`38h` 進 passthrough，前端讀運算元 1，值 0 就把畫面切到隊伍管理
（`programManaging`），B 或 ESC 回地圖並讓 ECL 從原地繼續。**從地圖進來的那次
不能走「開始冒險」那條路**——那會把開場整個重跑，隊伍被丟回起點，而畫面上
看起來只是「怎麼又在講故事」。

值 9 仍硬失敗：它多了兩件還沒讀出來的東西——問句的字串（`35DBh` 拿 `4948h`
當參數）與「答完之後讓 block 結束」的路徑。兩件都得靠猜才寫得出來。

其餘的值照原版直接繼續，什麼都不做。

驗收：`TestNormalKeysReachThePartyManagementCell` 只用按鍵走到 ECL3／block 11
那一格，開起畫面、按 B 回地圖，斷言位置沒變、開場沒有重跑、而且還走得動。
