# Spec 094：`3Bh SPELL`

狀態：READY（掃描範圍、寫回位置與停止條件都讀完）；
DRAFT（掃描起點比記憶陣列早七格，那七格是什麼還沒讀）。日期：2026-09-03。

## 它做的事（overlay-03 `2F64h`，3 個運算元）

```
2F72h  wanted = 運算元 1 的值            ; 法術編號
2F7Dh  slotAddr = 運算元 2 的位址
2F8Dh  memberAddr = 運算元 3 的位址
2F9Dh  i = 1；found = 0；members = 0
2FA5h  char = [5CF4h]
2FB1h  while char ≠ nil 且 found = 0：
2FBFh    if char[+17h + i] = wanted：found = 1；跳出
2FD8h    i ≤ 51h → i++ 繼續；否則 i = FFh、found = 1、跳出
2FF1h    char = char[+104h]；members++
3006h  寫入 slotAddr = i；寫入 memberAddr = members
```

## 兩個要照抄的地方

**外層迴圈實際上只跑得到第一個人。** 內層無論找到或掃完都會把 `found`
設成 1，所以外層在第一輪之後就退出。`members` 因此永遠是 1（隊伍非空時）。

**掃描起點是 `+17h + i`，i 從 1 起**，也就是記錄的 `+18h`；而記憶陣列
（spec 070）在 `+1Fh`。所以原版會先掃過七個不屬於陣列的位元組
（`+18h`..`+1Eh`），那七格是什麼還沒讀。

## remake 與原版的差異（明講）

remake 的角色沒有完整的 285-byte 記錄，只有記憶陣列本身，所以**只掃陣列
那 13 格**，槽位編號用 `k + (1Fh − 17h)` 換算回原版的編號。

差別是：原版那七格若剛好等於 `wanted`，原版會回報一個假的命中，remake 不會。
`+18h`..`+1Eh` 讀出來之前無法判斷那是不是原版的意圖。全遊戲只有一個呼叫點
（`ECL3/block 14 @5CAh`）。

比對前會濾掉第 7 位的旗標，與原版取名時的 `and al, 7Fh` 同一套。
