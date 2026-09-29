# Spec 165：事件的 `PICTURE`：HEAD／BODY 疊圖與 PIC 動畫

狀態：CONFORMED（`PICTURE` 的兩條路、畫在哪一框、`PICTURE 255` 收圖、神殿與商店門口
的肖像、PIC 動畫的張數與延遲、只在事件選單推動畫）；READY（腳本 `EXIT`／`COMBAT`
之後收圖——位元組是 exact，「接著由主迴圈重畫視野」是 strong inference）。
日期：2026-09-29（#120）。

spec 117 讀出 `PICTURE` 的分岔與 PIC 容器的版面，spec 164 把商店主選單那一框接上。
這一份把其餘場合（劇情事件、神殿、城區對話、碼頭……）接上，並補 PIC 那一條的動畫。
商店主選單那一框仍由 spec 164 管（`shopkeeperPortrait`）。

## 證據來源

| 輸入 | SHA-256 |
|---|---|
| `Pool of Radiance (1988).zip` | `1a7386c3…` |
| `poolrad/start.exe` | `12811cbc…` |
| overlay-03（ECL 分派器與各 opcode） | `5a3a18bd…` |
| overlay-04（神殿服務） | `d948ce6b…` |
| overlay-06（商店服務） | `2db20078…` |
| overlay-07（APPROACH／HEAD 疊圖、ECL 選單） | `a59f9d16…` |
| overlay-25（`COMBAT` 之後重畫） | `9fede24b…` |
| overlay-26（選單元件） | `cbdeafcf…` |
| overlay-27（重畫視野） | `8629cfbb…` |
| overlay-29（PIC／HEAD／BODY 載入與畫） | `f085c0ab…` |

overlay 由 `workplace/ovr/overlay-NN.bin` 取出（雜湊對 `docs/audit/dos-ovr-manifest.json`），
反組譯用 `coab-go-test:20260729` 的 GNU objdump 2.40（`-b binary -m i8086 -M intel`），
位址是 overlay 內的 code offset。far call 的段號照 `(executable_file_offset − 3B0h) ÷ 16`
換回 overlay：`0020h`＝overlay-03、`0045h`＝07、`010Ah`＝25、`011Dh`＝26、`0124h`＝27、
`012Bh`＝29。ECL 位址是 ECL3/0（`ecl3.dax` `db58f0c6…`，code base `9900h`）。

## `PICTURE` 的 handler（overlay-03 `0822h..091Ch`，exact）

分派表 `337Ah cmp ax,0Eh` → `3380h call 0822h`。運算元讀進 `[bp-1]`：

```
083B  les di,[4933h] ; cmp word es:[di+1CCh],0 ; jne 0873   ; ECL @49E6 == 0（野外）
084D  (3,3)-(13,13) 填 0 → call 150h:310h                  ; 先把框清黑
0873  cmp [bp-1],FFh ; je 08DF
0879  mov byte [828Fh],1 ; mov byte [82A6h],1              ; 「框裡是事件圖片」
0883  cmp word es:[([4937h])+5C2h],FFh ; jne 08CA           ; head 選擇子 = ECL @6DE1
0890  "PIC"（081Eh）, 運算元, 0, DS:6A1Ch → call 12Bh:39h    ; overlay-29 e5：載入 PIC 動畫
08B0  (3,3), 1, [6A22h:6A24h] → call 12Bh:2Ah               ; overlay-29 e2：畫第 1 張
08CA  (@6DE1, 運算元) → call 45h:48h                         ; overlay-07 e8 `051Eh`：HEAD／BODY
08DF  PICTURE 255：(495Ah>1 或 495Bh==1) 而且 (82A6h 或 82A9h) → call 124h:25h（重畫視野），
      清 82A6h／82A9h、立 82A7h；最後清 828Eh／828Fh
```

`(3,3)` 是 8 像素為單位，就是第一人稱內框的原點 `(24,24)`，88×88（spec 047／117）。
所以兩條路都**蓋滿第一人稱那一框**，右邊的隊伍欄照畫。

overlay-07 `051Eh` 先 `mov byte [82A7h],0`，再把 `(head, body)` 交給 overlay-29 e9 載入
`HEAD<區號>`／`BODY<區號>`，e8 以 `(3,3)` 畫出（spec 117）。

## 兩個場合的像素對照（exact）

| 場合 | ECL | 圖 | 基準 |
|---|---|---|---|
| 蘇恩神殿（地點分派索引 7，spec 102） | `A0BCh SAVE 22 → @6DE1`；`A0C2h PICTURE 24` | `HEAD3/22` 疊 `BODY3/24` | `docs/reference/original-dos/temple/64-y-sune-temple.png`（dosgolem `workplace/dosgolem-ref-temple/64-y`，SHA-256 `51d83458…`）|
| 菲蘭武具店門口 | `A8BFh SAVE 42 → @6DE1`；`A8C5h PICTURE 9` | `HEAD3/42` 疊 `BODY3/9` | spec 164（`dosgolem-ref-shop/78-Up` 與 `79-y` 那一框相同）|

神殿那一框從「DO YOU SEEK HEALING?」（`63-Up`）起就是祭司，服務選單（`64-y`）時還在。
負對照：另一座神殿 `A9DBh SAVE 57`／`A9E1h PICTURE 1` 的 `HEAD3/57 + BODY3/1` 對不上。
測試：`internal/assets` 的 `TestSuneTemplePriestessMatchesTheDOSTempleShot`。

## 圖什麼時候收掉

框裡的圖一直留著，直到視野被重畫。位元組看得到的有四處：

| 時機 | 位元組 | 證據等級 |
|---|---|---|
| `PICTURE 255` | overlay-03 `08DFh..0914h`：`call 124h:25h`（overlay-27 e1）重畫視野 | exact（重畫的條件 `495Ah`／`495Bh` 見〈不做〉）|
| `COMBAT` 回來之後（含神殿、商店服務與真的開打） | overlay-03 `19BBh` 清 `82A6h`，`19C0h call 10Ah:D9h`（overlay-25 e37）照 `DS:4954h`＝3／4 重畫視野（`28AAh..28D7h` 都是 `call 124h:25h`） | exact |
| 腳本 `EXIT` | overlay-03 `0090h` 清 `82A6h`；`@49E6 ≠ 0` 時 `0040h..0061h` 把 `(3,3)-(13,13)` 填黑 | exact |
| 走一步之後 | overlay-03 主迴圈 `391Ch call 124h:25h` 重畫視野，`3962h`／`3967h` 清 `82A6h`、立 `82A7h`，再跑下一支入口 | exact |

`EXIT` 之後「回到主迴圈、視野重畫出來」是 strong inference（`EXIT` 本身只清框與旗標）；
dosgolem `workplace/dosgolem-ref-spells` 的 `78-e`（店主）→ `79-e`（離店，視野）是對照。
腳本自己也幾乎都在 `EXIT` 之前寫 `PICTURE 255`（`9D50h`、`A1B0h`、`A2C2h`、`AA71h`……）。

## PIC 那一條的動畫（exact）

`PIC<區號>.DAX` 的一個區塊是一段動畫（spec 117〈PIC 容器的版面〉）。每一張前面的
**4 bytes 是延遲**（32-bit little-endian，選單元件以 `les ax,[di+6A16h]` 整個讀）：

| 區塊 | 內容 | 張數 | 延遲 |
|---|---|---:|---|
| `PIC3/41` | 碼頭的船 | 4 | 20／15／20／15 |
| `PIC3/29` | 營火（spec 135） | 2 | 2／2 |
| `PIC3/1` | 寶箱 | 4 | 2／4／6／1 |

overlay-29 e5 載入時把張數寫到 `DS:6A1Ch`、目前張號 `6A1Dh` 設 1（`01A8h`、`01D8h`）；
檔名是 `"PIC"`（`0099h`）而區塊是 1 時，張數大於 4 就截成 4（`01B5h..01CEh`）。

**換張只發生在選單元件等鍵的時候**（overlay-26 entry 3 `00E4h` 的迴圈，`024Ch..02C5h`）：
參數 `[bp+0Ah]` 不為 0 且 `6A1Dh > 0` 才畫第 `6A1Dh` 張、比時間；`經過的 BIOS tick >
延遲 ÷ 7`（`0288h mov cx,7` → `5BBh:279h` 除法，`02A1h..02A9h` 嚴格大於）就換下一張，
超過張數繞回 1。所以一張停 **`延遲 ÷ 7 + 1` 個 BIOS tick**：船是 3 個 tick（0.16 秒），
營火是 1 個。

誰傳「要動」：

| 呼叫端 | 傳的值 |
|---|---|
| `2Bh HORIZONTAL MENU`（overlay-03 `11B1h`，`1238h..124Ch`）| `82A6h && 82A7h` |
| `29h ENCOUNTER MENU`（overlay-03 `20B1h` → `1F87h`，`2200h..2213h`）| `82A6h && 82A7h` |
| `2Ch PARLAY`（overlay-03 `27A8h`，`280Dh`）| 0 |
| 神殿服務選單（overlay-04 `0D78h`）、商店主選單（overlay-06 `05FBh`）| 0 |
| 紮營的七層（overlay-15）| 1（營火，spec 135）|

HEAD／BODY 那一條在 `051Eh` 把 `82A7h` 清掉，所以肖像不會動；PIC 那一條沿用前一次的
`82A7h`（`PICTURE 255` 與主迴圈都把它立成 1）。

PIC 那一條在事件裡沒有 dosgolem 基準（碼頭要先買船票，路上有隨機遭遇）。同一個容器
與同一支解碼（`parseAnimationWithDelays`）已由營火逐格驗過（spec 135：區塊 29 第 1 張
7744/7744），所以船的像素是 strong inference；原版收據記在 `docs/audit/stop-line.md`。

## 契約

1. `applyCellECLResult` 遇到 `0Eh PICTURE` 時，`recordECLPicture` 記下
   `(@6DE1, 運算元)`（spec 164），`showEventPicture` 換成那一張；運算元 `FFh` 收圖。
2. 圖只在事件進行中（`cellEventPending`）才蓋在框上，Rolf 的 APPROACH 那一頁
   與導覽照 spec 117；`finishCellBlock`（腳本 `EXIT`）、`beginInitialSearch`（走完一步）、
   `enterCombatStaging`（`COMBAT`）與讀檔都收圖。
3. `@6DE1 ≠ FFh`：`loadNPCPortrait(區號, @6DE1, 運算元)`，一張不動。
   `@6DE1 = FFh`：`assets.ReadPICAnimation(區號, 運算元)`，每一張停
   `(延遲 ÷ 7 + 1) × 3` 個影格（一個 BIOS tick 取 3 影格，與營火相同），只在
   `cellWaitingMenu` 而且不是神殿、商店、WHO、戰利品選單時才換張。
4. 畫在第一人稱視野上面、同樣放大兩倍；右邊隊伍欄與文字框照畫。
5. 載不到就退回視野，不重試；換主題時丟掉快取重畫，張號留著。
6. 不看 `CAMP → ALTER → PICS`：handler 沒讀 `DS:4956h`（整批 overlay 裡讀它的只有
   overlay-19 `04F5h`）。商店主選單與 APPROACH 仍照 spec 164／117 的現況。

## 驗收

- `TestSuneTempleShowsThePriestessUntilTheScriptLeaves`：從 `Update()` 走進 (1,3)，
  YES／NO 那一頁與服務選單都畫 `HEAD3/22 + BODY3/24`、只載一次、不換張；選 Exit
  之後腳本 `PICTURE 255`，框裡回到視野。
- `TestTheBoatAnimatesWhileTheDockWaitsForReturn`：買票後走上碼頭 (15,1)，畫的是
  `PIC3/41` 四張的船，停在 `HORIZONTAL MENU` 時每 9 個影格換一張、繞回第 0 張；
  按 RETURN 上船換區之後收圖。
- `TestEventPictureOnlyAnimatesInEventMenus`：神殿選單不推、事件選單推、`EXIT` 與
  `PICTURE 255` 收圖。
- `TestPICAnimationCarriesItsDelays`、`TestPICAnimationMissingBlockFails`：延遲、張數、
  XOR 還原與缺區塊失敗。
- 發行包對拍 `temple` 那一張（`tools/appimage-dos-parity.sh`），數字記在 commit message。

## 不做

- `PICTURE 255` 在 `495Ah ≤ 1` 而且 `495Bh ≠ 1` 時不重畫視野（`08DFh..08EBh`）。
  兩個旗標的語意沒讀；remake 一律收圖。記在 `docs/audit/stop-line.md`。
- `@49E6 = 0`（野外）時 `PICTURE` 先把框清黑、`EXIT` 不清框：圖本身蓋滿同一塊，
  看不出差別。
