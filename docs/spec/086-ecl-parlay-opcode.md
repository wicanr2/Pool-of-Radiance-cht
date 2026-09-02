# Spec 086：`2Ch PARLAY`

狀態：READY（六個運算元、五個選項、結果表與寫回位址都讀完並實作）。
日期：2026-09-03。

## 形狀與 `29h ENCOUNTER MENU` 一樣

overlay-03 `27A8h`，六個運算元：

```
27AFh  取六個運算元
27BBh  for i = 0..4: v[i] = 運算元 (i+1) 的值        ; 五格結果表
27E2h  取字串 20h（`05BBh:074Fh`）
27F1h  用 cs:2785h 格式化（`05BBh:0634h`）
2801h  overlay-07 entry 21（`0045h:0089h`）顯示選單並取回選擇 → choice
2818h  dest = 運算元 6 的位址（`6E11h`／`6E51h` 是它的低／高位）
2828h  result = v[choice]
2835h  寫入 dest = result（overlay-07 entry 16，`0045h:0070h`）
```

所以：**運算元 1..5 是用選擇當索引的結果表，運算元 6 說結果寫進哪個 ECL
變數**。remake 不必解讀那些碼的意義，寫對就好，分支由 ECL 自己做。

## 五個選項

`cs:2785h` 是 `~HAUGHTY ~SLY ~NICE ~MEEK ~ABUSIVE`——`~` 標的是熱鍵字母，
與 spec 078 的 `~COMBAT ~WAIT ~FLEE ~PARLAY` 同一套寫法。順序即結果表索引。

這五個字來自 **overlay-03 的程式碼字串**而不是 ECL 文字，所以中譯歸 remake
的 UI 訊息表管，不進 `internal/gametext` 的原文對照（那一份只收 ECL 文字，
而且有測試盯著每一條原文都要在盤點檔裡）。

## remake 怎麼接

`2Ch` 走 passthrough，前端解出結果表與寫回位址，把五個語氣擺上既有的
cell 選單，玩家按 ENTER 之後把那一格的碼寫進 `resultAddress` 再讓 ECL 繼續。

位址一定要用 `WordAddress` 取——`NumericValue` 會把變數的**內容**當成位址，
那個錯誤在測試裡看起來像「寫進去了」，只是寫錯地方。

## 還沒讀

- `05BBh:074Fh` 取的字串 20h 是什麼（選單標題？）。remake 目前用自己的提示句。
- 五個結果碼在各個呼叫點分別代表什麼——不需要知道也接得起來，
  但寫劇情文件時會需要。
