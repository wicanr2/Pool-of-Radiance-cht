# Spec 090：`39h WHO` 與「目前角色」

狀態：READY（挑人、寫進「目前角色」、以及誰會讀它都讀完並實作）；
DRAFT（提示字串的來源、overlay-25 entry 42 的挑人範圍）。日期：2026-09-03。

## `DS:5CF0h` 是「目前這個角色」

四條 opcode 圍著同一個 far pointer 轉（spec 083 先記過前三條）：

| opcode | 對 `5CF0h` 做什麼 |
|---|---|
| `39h WHO` | **寫**：把玩家挑到的角色放進去 |
| `28h ROB` | **讀**：運算元 1 為 0 時只搜那個人的身 |
| `36h ADD NPC` | **讀**：把欄位寫進它指到的記錄 |
| `38h PROGRAM` | 在 `3174h` 從 `4394h` 搬進去 |

## `39h WHO`（overlay-03 `2E90h`，1 個運算元）

```
2E97h  取運算元 1                       ; 之後沒看到它被讀
2EA5h  overlay-37 entry 14，參數 (1, 11h, 26h, 16h)   ; 清畫面下半的提示框
2EB0h  從 DS:6E8Eh 讀提示字串到區域緩衝區
2ECDh  overlay-25 entry 42（`2C81h`），參數是 &DS:5CF0h  ; 出參
```

`DS:6E8Eh` **在 START.EXE 的檔案映像之外**（檔案 47,936 bytes，該位址算出來
是 58,942），也就是 BSS，執行時才填——提示字串靜態讀不到。remake 用自己的
一句「誰？」，並在 spec 裡標明那不是原文。

## 挑完人要把他投影進視窗

`5CF0h` 是指標，`6B00h` 是**視窗**——ECL 讀的是視窗（spec 021）。所以
`39h` 改了指標之後，視窗必須跟著換人，否則後面的腳本讀到的還是上一個人。

港務長那一段就是這個形狀（ecl3/0 `A1B4h`）：

```
A1B4  WHO 'WHO WILL PAY?'
A1C1  COMPARE @6BB8, 128 ; IF < ; GOTO A1E5     ; 士氣：NPC 不肯付
A1CC  PRINTCLEAR "THAT PERSON WON'T PAY."
A1E5  COMPARE 1, @6BC3 ; IF > ; GOTO A20D       ; 白金不夠
A1F0  SUBTRACT 1, @6BC3 → @6BC3                 ; 扣一枚
A1F9  ADD 128, @6DB4 → @6E79
A202  LOAD CHARACTER @6E79                      ; +128 是重畫那個人的面板
A20D  PRINTCLEAR "YOU DON'T HAVE ENOUGH PLATINUM."
```

**`6BC3h` 是那個人身上的白金。** 賭場那一段把它印出來當餘額
（ecl3/0 `A513h`：`SAVE @6BC3 → @4A1A`、`PRINTCLEAR 'YOU HAVE'`、`PRINT @4A1A`、
`PRINT 'PP.'`），贏了再 `A62Fh SAVE @4A1A → @6BC3` 寫回去。全部 29 個區塊裡
碰 `6BC3h` 的只有這兩段（六處）。

**錢不必寫回去，因為視窗就是本尊。** `5CF0h` 是指標，指到那個人的 285-byte
記錄（spec 021），所以 `SUBTRACT 1 → @6BC3` 改的就是那個人身上的白金。
`0Ah LOAD CHARACTER` 的運算元加 128 是另一件事：高位設起來（而且
`DS:8298h`／`DS:8299h` 都非零）才走 `036Dh..03A8h` 的 legacy buffer／重畫路徑
（spec 021 第 4 點）——腳本改完記錄之後叫它，是要那個人的面板重畫，
選人那一半照樣會做。

> `6BC3h` 相對於視窗起點是 `+C3h`，與 spec 040 記的「角色 record `+88h + 2*i`
> 七種貨幣」對不上（白金依那個版面應在 `+90h`）。兩者的關係還沒讀出來；
> 這裡只依 ECL 自己的用法認定 `6BC3h` 是白金，不改寫 spec 040 的版面。

## remake 怎麼接

`39h` 走 passthrough，前端把隊伍成員的名字擺上既有的 cell 選單，
ENTER 之後把索引記進 `currentCharacter`，**並把那個人投影進視窗**
（`gamepack.ProjectCharacterWindow`），再讓 ECL 繼續。
`28h ROB` 的範圍 0 讀的就是這個索引。

remake 的記憶體是一張 map，沒有「視窗疊在記錄上」這回事，所以拿
**選進來時抄進去、換人之前抄回來**補上：`gamepack.CharacterBinding` 記住視窗
裡現在是誰，`Character(index)` 供投影，`CommitPlatinum(index, value)` 在換人
之前與腳本收尾（`Flush`）時把值抄回去。`cmd/pool-game` 的 window 直接讀寫
`state.Party`，所以船資扣掉的、賭場贏來的白金都留在存檔裡。掃描與測試用的
session 走 `staticCharacterWindow`，投影得出去、抄回來是空操作。

## 還沒驗

- 賭場那一段（ecl3/0 `A4F2h`，城區 terrain 索引 20）是視窗的第二個消費者，
  而且是**加錢**的方向（`A62Fh SAVE @4A1A → @6BC3`）。目前只有港務長扣錢那一
  條有測試釘著。

## 還沒讀

- overlay-25 entry 42（`2C81h`）挑人的範圍：只有隊伍，還是含 NPC 與倒下的人。
- `DS:6E8Eh` 的提示字串由誰填。
- 運算元 1 的用途（被取出來但沒看到被讀，可能由 entry 42 自己取）。
