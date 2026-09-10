# Spec 117：APPROACH 的 NPC 半身像

狀態：READY（`SETUP MONSTER` 三個 operand、HEAD／BODY 檔名與疊法、畫的位置、
Rolf 的 head 區塊）；DRAFT（head 選擇子 `[+5C2h]` 的 producer）。
日期：2026-09-05。

## 為什麼會找上這一段

`tools/capture-dos-adventure.sh` 把原版走到第一人稱畫面之後，
`03-rolf-approach.png` 顯示的是**半身像整個蓋住那一框**，不是第一人稱視野。
remake 那時仍畫視野，所以這是對拍差異裡最顯眼的一項
（`docs/reference/original-dos/adventure/README.md`）。

Spec 010 原本把「Rolf APPROACH 圖像」列成 DRAFT，理由是
「在 PIC／sprite consumer 尚未閉合前，不授權 remake 猜一張 Rolf 圖」。
現在有原版畫面可以逐格比對，不必猜。

## 原版怎麼走（exact）

ECL3/block 0 的 Rolf handler 在 `B0AAh` 執行 `SETUP MONSTER 12, 2, 9`，
接著兩個 `APPROACH`（`B0B2h`／`B0B4h`）。

overlay-03 的 dispatcher 把三個 opcode 分給三支 handler：

| opcode | 名稱 | overlay-03 |
|---|---|---|
| `0Ch` | SETUP MONSTER | `03B1h` |
| `0Dh` | APPROACH | `07E1h` |
| `0Eh` | PICTURE | `0822h` |

`03B1h` 把三個 operand 分別存到 `DS:6D47h`、`ES:[DI+580h]`（`DS:4937h` 那個
結構）與 `DS:6D48h`，再由 `DS:6A0Bh..6A0Dh` 算出實際距離寫進 `+582h`，
最後把 `(6D47h, 6D48h, +582h, DS:828Eh)` 交給 `0045h:004Dh`。
`07E1h`（APPROACH）先把 `+582h` 減一，再交給同一支。

`0045h` 依 spec 109 是 **overlay-7**，`004Dh` 是它的第 9 個進入點，
code offset `055Dh`。那支的分工是：

1. 第一個 operand（12）配 overlay-7 `0553h` 的 Pascal 字串 `"SPRIT"`，
   載入 `SPRIT<區號>.DAX` 的區塊 12——那是第一人稱視野裡走過來的人形。
2. `+5C2h` 是 **head 選擇子**。是 `0FFh` 就走 `"PIC"` 那條
   （`PIC<區號>.DAX` 的區塊 = 第三個 operand，單張 88×88）；
   不是 `0FFh` 就走 `0699h` 的 head／body 那條。
3. head／body 那條呼叫 overlay-7 `051Eh`，它把 `(+5C2h, 第三個 operand)`
   交給 `012Bh:004Dh` ＝ **overlay-29 進入點 9**（code offset `05A4h`）。
   那支用 `059Ah` 的 `"HEAD"` 與 `059Fh` 的 `"BODY"` 兩個字串各接上
   `DS:52D4h`（現行區號）組檔名，分別載入 head 與 body 區塊並各自快取。
4. 接著呼叫 overlay-29 進入點 8（`0662h`），參數 `(1, 3, 3)`：
   先畫 head、再畫 body，位置 `(3,3)`。**那是 8 像素為單位的座標**，
   換算就是 `(24,24)`——正好是第一人稱內框的原點（spec 047）。

所以 `SETUP MONSTER a, b, c` 的語意是：`a` = `SPRIT` 區塊、`b` = 接近距離、
`c` = `BODY` 區塊。ECL 的 `PICTURE n` 走的是同一個 `+5C2h` 分岔：
有 head 就畫 `BODY` 區塊 n，沒有就畫 `PIC` 區塊 n。

**一個順帶的自洽檢查**：ECL3/block 0 出現的 `PICTURE` operand 是
1、2、4、7、9、24、34、41、255。`PIC3.DAX` 只有區塊 1、29、41；
`BODY3.DAX` 有 1、2、4、7、9、24、34。兩邊各自蓋住不同的一組，
沒有一個 operand 落在兩邊都沒有的位置。

## Rolf 的 head 區塊（量出來的）

`+5C2h` 的 producer **在 ECL 裡**（2026-09-10 找到）。38 顆 overlay 與
`START.EXE` 逐位元組掃過 `5C2h` 這個位移，唯一的寫入是 overlay-7 `0233h` 在
ECL block 初始化時寫 `0FFh`——當時的結論「這是掃描面的洞，不是『沒有
producer』」是對的，洞在於**那種掃描看的是 overlay 的定址形式，而這個位址是
ECL 變數**。

`DS:4937h` 是 class 1 的基底，換算式 `[4937h] + 2A00h + addr × 2`（spec
008／106）把 `+5C2h` 換回 ECL 位址 **`6DE1h`**。
`cmd/pool-ecl-memory-audit -addresses 6DE1` 在八個 ECL 檔裡找到 **212 個
`SAVE`**。

換算有兩個獨立佐證：同一式子把 `+5C4h` 換成 `6DE2h`，而 remake 的蘇恩神殿
服務票（spec 017）本來就在讀寫 `Memory[0x6DE2]`；把 class 0 的 `+1CCh` 換成
`49E6h`，與 spec 074 獨立讀出來的值相同。

**還沒對的是值的語意**：那 212 處各寫什麼、與這裡量出來的 head 區塊 8 對不
對得上，要把 `SAVE` 的另一個運算元讀出來才知道。下面那段「從畫面量出來」的
證據不受影響——它本來就是獨立的一條路。

因此 head 是從原版畫面量出來的：`03-rolf-approach.png` 裁下 `(24,24)` 起的
88×88，上半 88×40 與 `HEAD3.DAX` 區塊 **8** 逐格 100% 相同，下半 88×48 與
`BODY3.DAX` 區塊 **9** 逐格 100% 相同（後者正好等於 ECL 的第三個 operand）。

負對照：八個 `PIC*.DAX` 的每一張 88×88（用「1 byte 張數＋每張 4 byte 前綴＋
17 byte 圖片頭」的版面解出來）與同一塊截圖比對，最高不到 60%。所以那張圖
不可能來自 `PIC` 那條路，head／body 那條是唯一解。

## PIC 容器的版面（順手解出來的）

`PIC*.DAX` 的區塊不是單張圖，是**一段動畫**：

```
+0        張數
每一張：  4 bytes 前綴 ＋ 17 bytes 標準圖片頭 ＋ 像素
```

`PIC3.DAX` 區塊 1 是四張的寶箱、區塊 41 是四張的船。第二張以後是與第一張
XOR 的差分（overlay-29 `0338h..03B7h` 那段 XOR 迴圈只在檔名是 `PIC` 或
`FINAL` 時才跑，`0101h..0129h` 就是在比這兩個字串）。

## 契約

1. `InitialEvent` 保存 `SETUP MONSTER` 的三個 operand
   （`SpriteBlock`／`ApproachDistance`／`PortraitBody`），不再只留第一個。
2. `assets.ReadNPCPortrait(zip, 區號, head, body)` 讀
   `HEAD<區號>.DAX` 與 `BODY<區號>.DAX` 的指定區塊，head 在上、body 在下疊成
   88×88；形狀不是 88×40／88×48 就失敗，不補零。
3. 正常路徑在 Rolf 的第一頁把那張圖蓋滿第一人稱內框；導覽開始之後不蓋
   （原版的 `05-tyr-stop-11-2-south.png` 那一框是視野）。
4. head 區塊以 `gamepack.RolfPortraitHeadBlock` 宣告，並在該處寫明它是量出來的、
   producer 未閉合。找到 producer 之前不要把它推廣到別的 NPC。
5. 疊不出來就退回畫視野；事件本身不得被一張圖擋住。

## 驗收

- `TestNPCPortraitMatchesTheDOSApproachShot`：疊出來的 88×88 與
  `03-rolf-approach.png` 那一框逐格相同。head 區塊、body 區塊、上下順序
  任何一項換掉都會有像素對不上。
- `TestReadDOSInitialEventMatchesSpec010`：三個 operand 是 `12, 2, 9`。
- `TestApproachPortraitOnlyCoversTheGreetingPage`：只有第一頁蓋、會快取、
  換主題會重疊。
- `TestApproachPortraitFallsBackToTheView`：疊不出來就退回視野，而且不重試。

## 仍未閉合

- `+5C2h` 的 producer（見上）。找到之前，別的 NPC 的 head 只能同樣靠對拍。
- `SPRIT<區號>.DAX` 的區塊還沒畫進第一人稱視野；`APPROACH` 的距離遞減與
  「走近」的動畫也還沒接。
- overlay-7 `055Dh` 裡 `DS:495Bh`、`DS:82AFh`、`DS:4957h` 三個旗標控制的分支
  尚未讀完，所以「什麼時候才會蓋半身像」目前是照對拍結果接的，不是照旗標接的。
