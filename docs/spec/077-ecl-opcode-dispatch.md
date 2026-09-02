# Spec 077：ECL 指令的派發鏈與運算元個數

狀態：CONFORMED（55 條 opcode 的處理常式位移與運算元個數，直接從 overlay-03
的位元組量出來）；DRAFT（`0x34 ECL CLOCK` 的個數與共用 engine 的表不同、
`0x29 ENCOUNTER MENU` 的語意）。日期：2026-09-03。

## 派發鏈長什麼樣

overlay-03 的直譯器用一長串比較做派發：

```
346E  3D 27 00        cmp  ax, 27h
3471  75 07           jne  下一個
3473  0E              push cs
3474  E8 0A E6        call 1A81h          ; TREASURE
3477  E9 5D 01        jmp  收尾
```

兩條 opcode 共用一支常式時前面多一個 `je`（`25h`／`26h` 就是這樣）。
整條鏈解出 **55 條**。

每支常式開頭都用同一個樣式取運算元：

```
1A88  B0 08           mov  al, 8
1A8A  50              push ax
1A8B  9A 2A 00 45 00  lcall 0045h:002Ah   ; 取 8 個運算元
```

`al` 就是個數。**掃描的上界要用「下一支常式的起點」夾住**——少了那一道，
掃描會越過 `retf` 讀到下一支常式的序言，量出來的數字看起來很合理但是錯的
（`31h SPRITE OFF` 與 `3Dh CLEAR BOX` 都會被誤判成 1）。

整張表在
[`docs/audit/pool-ecl-opcode-operands.json`](../audit/pool-ecl-opcode-operands.json)。

## 與共用 engine 的表比對

engine 的 `ecl.KnownCommands` 註明來源是「公開的 ECL dump 表與 CoAB 的重製」，
是二手的。這裡量的是 Pool 自己的位元組。逐條比對之後：

| 情況 | 條數 |
|---|---:|
| 相符 | 50 |
| 長度可變、engine 記 0（量到的是固定前綴）| 4 |
| 真的不一致 | 1 |

長度可變的四條是 `15h VERTICAL MENU`（前綴 3）、`25h ON GOTO`（2）、
`26h ON GOSUB`（2）、`2Bh HORIZONTAL MENU`（2）。engine 對這四條記 0 並改用
`RecordEnd` 算結尾，所以不算衝突——量到的前綴反而是補充資訊。

**唯一真的不一致的是 `34h ECL CLOCK`**：Pool 的常式（`2E1Ah`）只取一個運算元，
engine 的表寫 2。arity 錯一個 byte，直譯器每遇到一條就會往後多走一格，而且
**不會報錯**，只會安靜地把後面的指令流讀偏。engine 同時服務《青色枷的詛咒》，
所以在 CoAB 那邊也量過之前先不動它；現況鎖在
`TestECLOpcodeTableAgreesWithTheEngineExceptTheKnownGaps` 裡，別讓它悄悄變。

## `29h ENCOUNTER MENU` 是目前主線的攔路石

只用按鍵從標題走到索寇要塞（archive 4、block 21）之後，ECL 停在

```
continue Pool SearchLocation: opcode 0x29 at 1288 has no core handler or adapter passthrough
```

它的常式在 `20B1h`，取 **14 個運算元**（與 engine 的表一致），開頭就
`mov byte ptr ds:4961h, 1` 並開一個 0x450 bytes 的堆疊框，接著把運算元
1、2、3 分別寫進 `ds:6D47h`、某筆記錄的 `+580h` 與 `ds:6D48h`，再用
運算元 5..9 填一個五格陣列。語意還沒讀完，所以**不要**先把它宣告成
passthrough 讓它靜靜跳過——跳過一個 encounter 與正確執行它在報表上分不出來。
