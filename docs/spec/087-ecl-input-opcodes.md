# Spec 087：`0Fh INPUT NUMBER` 與 `10h INPUT STRING`

狀態：READY（兩條的寫回位址、長度上限與空字串處理都讀完並實作）。
日期：2026-09-03。

兩條都吃兩個運算元，都把玩家打的東西寫進**運算元 2** 指的 ECL 變數。
提示句不由它們帶，由前一則 `12h PRINT` 給——所以 remake 只要把輸入列擺在
既有的文字框下面就好，不必發明提示文字。

## `0Fh INPUT NUMBER`（overlay-03 `091Dh`）

```
0923h  取兩個運算元
092Fh  dest = 運算元 2 的位址（`6E0Dh`／`6E4Dh` 是它的低／高位）
093Fh  overlay-37 entry 8（`0198h:0048h`），參數 (0Ah, &buffer) → ax
094Fh  寫入 dest = ax
```

## `10h INPUT STRING`（overlay-03 `0960h`）

```
0967h  取兩個運算元
0973h  dest = 運算元 2 的位址
0983h  overlay-37 entry 7（`0198h:0043h`），參數 (28h, 0Ah, &len, &buffer)
       → 最長 40 個字元
09A8h  buffer 是空的時候，改塞 `cs:95Eh`——長度 1 的一個空白
09C3h  overlay-07 entry 18（`0045h:007Ah`）把字串寫進 dest
```

空字串換成一個空白這件事**要照做**：ECL 後面會拿那個變數去比字串
（共用 VM 的 `03h` 在任一邊是字串時走字串比較），「什麼都沒打」與
「打了一個空白」比出來的結果不同。

## remake 怎麼接

兩條走 passthrough。前端解出運算元 2 的位址（要用 `WordAddress`），
擺出輸入列，收鍵盤：數字只收 `0`..`9`，字串收可見字元並轉大寫，
兩者都在 40 個字元停下來。ENTER 之後：

- `0Fh` 寫 `Machine.Memory[dest]`（沒打就是 0，不是錯誤）。
- `10h` 寫 `Machine.Strings[dest]`——共用 VM 的字串變數是另一張表，
  操作碼 `81h` 的運算元從那裡讀。

輸入列吃掉整個影格：打字的時候 ESC 與方向鍵不該有別的意思。

## 玩家要打哪些字（密語盤點）

`10h` 一共 **20 處**，盤點在 [`docs/audit/dos-password-prompts.json`](../audit/dos-password-prompts.json)，
由 `cmd/pool-password-audit` 產生（`tools/go.sh run ./cmd/pool-password-audit > docs/audit/dos-password-prompts.json`）。
每一處記檔名、block、位址、寫回的變數、問句與答案。

答案怎麼解由 `gamepack.InputAnswer` 決定，**遊戲裡顯示密語提示（spec 141）走的是同一支**，
所以盤點檔綠不綠就是提示準不準。解法是從輸入的下一條往後掃 64 條，找比對這個變數的
`03h COMPARE`；比的是變數時，把沿路 `09h SAVE <字面>` 寫進那個變數的字面都收起來。

三個會產生「自洽但錯」結果的地方，各由一處真實資料釘住：

| 坑 | 會發生什麼 | 釘住它的那一處 |
|---|---|---|
| `COMPARE` 兩個運算元的次序不固定 | 只認 `COMPARE <變數> <字面>` 時，13／20 解不出來，而解不出來長得像「這一處沒有答案」 | ecl2/9 `9F1C`：`COMPARE "HARASH" 9836h` |
| 直線往下讀會走進不相干的下一段 | 巨人那一處撿到隔壁的 `TYRANTHRAXUS`，在遊戲裡是個看起來很像真的假答案 | ecl5/5 `9E08`：後面接 `RANDOM` 挑一句罵人的話再 `GOTO` |
| 緊接在 `IF` 後面的 `GOTO` 是有條件的 | 跟著跳過去會把答案掃丟（另一半的路徑才是答案） | ecl7/23 `A4C6h` 之後才是 `COMPARE 6E79h "NOKNOK"` |

目前 19／20 解得出來。剩下那一處（ecl5/5 `9E08h`，巨人問口令）**原版就沒有答案**：
打什麼都是 `'WRONG!'`，`RANDOM` 決定罵哪一句，所以提示不出現是對的。

解出來的密語：`HARASH`、`TYRANTHRAXUS`、`OHLO`、`SAMOSUD`／`SHESTNI`、`LUX`、
`RHODIA`、`NOKNOK`、`SAVIOR`。

**指令表要帶著走**：掃描用的 `decode` 一定要帶 `gamepack.PoolCommandTable()`。退回 engine
底稿時 `34h ECL CLOCK` 的運算元個數是 2 不是 1，掃過它之後每一條都錯位（spec 093）。
`cmd/pool-game` 的 `eclInstruction` 2026-09-18 之前就是退回底稿的，已改。

## 還沒讀

- `0198h:0048h` 的參數 `0Ah` 是最大位數還是欄位寬度。
- `0198h:0043h` 的第二個參數 `0Ah`（`28h` 已確認是最大長度）。
- 運算元 1 在兩條裡都被取出來但沒被用到，它的用途未知。
