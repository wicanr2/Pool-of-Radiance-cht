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

**錯誤只有一種，67 次：`LOAD CHARACTER ... requires a title projector`**
（ecl5/7 2 次、ecl7/22 44 次、ecl7/23 20 次、ecl8/13 1 次）。
其餘 26557 次入口執行都停在正常邊界（exit／event／menu），沒有解碼失敗、
沒有未知 opcode、沒有無窮迴圈。

四個區塊的每一格都停在 `2Dh CALL`：**ecl6/25、ecl7/26、ecl8/27、ecl8/29**。
前三個正是 spec 101 認出來的樞紐。ecl1/18 則是每一格都停在選單。

前端事件的分佈（次數由多到少）：`31h SPRITE OFF`、`12h NEWECL`、
`0Eh PICTURE`、`2Dh CALL`、`3Ah`、`1Eh CHECKPARTY`、`22h PARTY SURPRISE`、
`0Dh APPROACH`、`11h`、`21h LOAD FILES`。

## 唯一的缺口：LOAD CHARACTER 的投影器

`1Ch LOAD CHARACTER` 要有一個把角色記錄投影進 VM 的函式；目前只有開場那條路
接了（`SetCharacterProjector`）。掃描用的乾淨 session 沒接，所以那 67 次會報
錯。**這不是四個區域專有的問題**——是那四個區域的腳本會叫 NPC 加入。

## OPEN

- `3Ah` 與 `11h` 兩個 opcode 的語意（掃描裡都出現了，還沒逐條讀）。
- 掃描沒有辦法回答「這一格在正常遊玩時會發生什麼」：變數全是 0，
  分支會走進與實際遊玩不同的支線。要那個答案得走 spec 101 的路。
