# Spec 103：整包格子入口掃描

狀態：READY（工具與量到的數字）

## 這一條回答什麼

`cmd/pool-world-cell-sweep` 把**每一張地圖的每一格、四個朝向**都餵進它自己的
ECL 區塊，跑入口 0（移動前）再跑入口 1（移動後），記下停在哪一種邊界。

```
tools/go.sh run ./cmd/pool-world-cell-sweep -text
```

它**不是**「玩家走得到」的證明——每個區塊都從乾淨的變數開始，沒有走過主線。
它回答的是另一個問題：**remake 的 VM 撐不撐得住整包遊戲的格子腳本**，
以及剩下的世界會要求哪些前端事件。走得到多少是 spec 101 的事。

## 量到的（26 個區塊、26624 次入口執行）

有地圖的 ECL 區塊有 26 個（另外三個——ecl3/8、ecl3/11、ecl6/19——是室內
場景，沒有格子）。GEO 的 30、31、32 沒有自己的 ECL，由別的區塊 `LOAD FILES`
帶進來（spec 101 的表）。

**26624 次入口執行，0 個錯誤。** 全部停在正常邊界（exit／event／menu），
沒有解碼失敗、沒有未知 opcode、沒有無窮迴圈。

四個區塊的每一格都停在 `2Dh CALL`：**ecl6/25、ecl7/26、ecl8/27、ecl8/29**。
前三個正是 spec 101 認出來的樞紐。ecl1/18 則是每一格都停在選單。

前端事件的分佈（次數由多到少）：`31h SPRITE OFF`、`12h NEWECL`、
`0Eh PICTURE`、`2Dh CALL`、`3Ah`、`1Eh CHECKPARTY`、`22h PARTY SURPRISE`、
`0Dh APPROACH`、`11h`、`21h LOAD FILES`。

## 掃描要帶一支隊伍

`1Ch LOAD CHARACTER` 與 `1Dh PARTYSTRENGTH` 要有投影器才答得出來，正常遊玩
那條路（`NewInitialEventSession`）本來就會接。掃描用的乾淨 session 一開始沒
接，量出 67 次「requires a title projector」——**那是掃描的環境問題，不是腳本
的問題**。接上六個人之後歸零。

## OPEN

- `3Ah` 與 `11h` 兩個 opcode 的語意（掃描裡都出現了，還沒逐條讀）。
- 掃描沒有辦法回答「這一格在正常遊玩時會發生什麼」：變數全是 0，
  分支會走進與實際遊玩不同的支線。要那個答案得走 spec 101 的路。
