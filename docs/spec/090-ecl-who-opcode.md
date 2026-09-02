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

## remake 怎麼接

`39h` 走 passthrough，前端把隊伍成員的名字擺上既有的 cell 選單，
ENTER 之後把索引記進 `currentCharacter`，再讓 ECL 繼續。
`28h ROB` 的範圍 0 讀的就是這個索引。

## 還沒讀

- overlay-25 entry 42（`2C81h`）挑人的範圍：只有隊伍，還是含 NPC 與倒下的人。
- `DS:6E8Eh` 的提示字串由誰填。
- 運算元 1 的用途（被取出來但沒看到被讀，可能由 entry 42 自己取）。
