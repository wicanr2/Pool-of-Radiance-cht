# Spec 108：結局過場（overlay-18 entry 1）

狀態：READY（進入點、十三行字、圖序、版面座標）；DRAFT（逐格動畫的節奏、
`FINAL` 那幾張圖的解碼與調色）。日期：2026-09-03。

## 一句話

打贏泰倫斯拉克斯之後，`ECL5/7` 的 `A82Ah PROGRAM 08` 會叫出 overlay-18
entry 1（`02A1h`）——那是整部遊戲的結局：龍倒下、泰倫斯拉克斯的靈魂從屍體
裡竄出、最後被吸回輝光之池。remake 目前把這一段跳過去。

## 怎麼走到這裡

```
a7fa  CLEARMONSTERS
a7fb  LOAD MONSTER 42h, 1, 42h      ; mon5/66 TYRANITHRAXUS，HP 80 AC 0
a802  COMBAT
a803  COMPARE @6DC7, 81h ; IF = ; GOTO A5EA   ; 沒打贏那一支
a80e  COMPARE @4ABA, FFh ; IF <>
a815  SAVE FEh → @4ABA               ; 市政廳槽 20「YOUR QUEST IS OVER」
a81b  OR @4A6D, 10h → @4A72
a824  SAVE 01 → @4AE0
a82a  PROGRAM 08                     ; ← 這裡
a82d  PRINTCLEAR "KNOWING THAT TYRANTHRAXUS HAS FINALLY BEEN DEFEATED, ..."
a8e5  座標設回 (0,4) 朝向 1、6E12 = 3、NEWECL 0
```

`ECL5/7` 是三個靜態展開解不開的區塊之一，所以上面的位址是用線性指令掃描
（`ecl.ScanKnownInstructions`）讀出來的。spec 081 先前寫「值 8 全遊戲沒有
呼叫點」，就是因為只掃了走得到的碼。

## overlay-18 entry 1 在做什麼

```
02a8  用 DS:52D4h（目前 archive，這裡是 5）組出檔名放進 DS:4675h
02c2  存下 DS:4954h，設成 7；清 4670h/4672h
02d5  0198h:0066h(16h, 26h, 11h, 1)        ; 開一個文字框
02e6  0198h:002Fh(0111h, 0Ah, 11h, 1)      ; 逐行印
0303  0198h:002Fh(0135h, 0Ah, 12h, 1)
0320  0198h:002Fh(015Ah, 0Ah, 13h, 1)
033d  010Ah:00D9h()                        ; 等一個鍵
0342  "FINAL" ＋ DS:4675h → 018Eh:0039h(區塊 1) → 018Eh:004Dh 畫出來
03d1  同上，區塊 3
042b  同上，區塊 4（前置 [4937h]+67Ch > 1）
0486  同上，區塊 5（前置 > 2）
04e2  同上，區塊 6（前置 > 3）
0517  再開文字框，印 0178h/019Ch/01C1h/01E8h/0208h/022Dh
      與 0236h/025Dh/0285h/0298h
```

檔名組出來是 **`FINAL5.DAX`**，ZIP 裡確實有 `poolrad/final5.dax`
（36,603 bytes、8 個區塊），這是「檔名由 archive 編號組成」的正對照。

## 十三行字（逐字取自 overlay-18 `0111h..02A0h`）

| 位移 | 列 | 原文 |
|---|---:|---|
| `0111h` | 17 | `Mortally wounded, the dragon roars!` |
| `0135h` | 18 | `The spirit of Tyranthraxus flares up` |
| `015Ah` | 19 | `from the dragon's body.` |
| `0178h` | 17 | `"Fools, you have but slain the body` |
| `019Ch` | 18 | `I possessed.  I cannot be defeated!"` |
| `01C1h` | 19 | `With the power of the pool of radiance` |
| `01E8h` | 20 | `which I moved here, to my lair,` |
| `0208h` | 21 | `I will still rule, by possessing one` |
| `022Dh` | 22 | `of you!"` |
| `0236h` | 17 | `"No Lord Bane!  I can still rule here!` |
| `025Dh` | 18 | `I have not failed.  Do not call me back` |
| `0285h` | 19 | `through the pool!"` |
| `0298h` | 20 | `Noooo...` |

`0172h` 的 `FINAL` 不是台詞，是拿去組檔名的。

## remake 現況與驗收

`internal/gamepack.ReadDOSEndingScript` 從原版 ZIP 取出這十三行與圖序，
`TestEndingScriptMatchesTheOriginalOverlay` 逐字釘住。

**台詞已經接上**：`38h PROGRAM` 的值 8 進 `enterEnding`，一頁一個 ENTER，
翻完之後讓 ECL 往下跑到 `A82Dh` 的敘述文字。分頁是資料算出來的——原版每開
一次文字框就從第 `11h` 列重新印起，所以**列號回到第一列就是新的一頁**，
三頁分別是 3、6、4 行。`TestDefeatingTyranthraxusSetsTheVictoryFlag` 從打贏
一路驗到翻完三頁。

**中譯也接上**：十三行進了 `internal/gametext/zh-TW.json`（共 1,744 條）。
它們是 overlay 內嵌字串、不在 ECL 的 6-bit packed 盤點裡，所以
`TestEverySourceExistsInTheOriginalInventory` 改成兩個來源都認——守的還是
同一件事（每一條原文都要真的出現在原版資料裡），只是原版的字本來就有兩種
存法。`TestEndingCutscenePagesAreTranslated` 擋「翻了但沒接上」。

**圖還沒畫**：`FINAL5.DAX` 的區塊 1、3、4、5、6 解得出 120×120、120×120、
56×32、16×48、16×48（`graphics.ParsePicture`，同 `TITLE.DAX` 那條路）。
兩支常式已經由 spec 109 的對照表查出來：`018Eh` 是 **overlay-36**，
載入是 entry 5（`011Bh`、stub `39h`）、繪製是 entry 9（`0D9Fh`、stub `4Dh`）。

### 繪製常式的引數（2026-09-04 讀出來）

`0D9Fh` 是 Turbo Pascal 的 `push bp / mov bp,sp`，引數由左而右推，
所以最後推的落在 `[bp+6]`：

| 位移 | 呼叫端推的 | 用途 |
|---|---|---|
| `[bp+14h]` | `0` | **X**：負數取絕對值放 `[bp-2]`，非負放 `[bp-6]`；接著與圖片記錄 `+2`（寬）比大小，超過就直接返回 |
| `[bp+12h]` | `0` | Y（同一組比較走 `[bp+6]` 那一邊的 `+2`）|
| `[bp+10h]` | `0` | 還沒讀 |
| `[bp+0Eh]` | `0` | **旗標**：位元 0 與位元 2 各控一件事（位元 0 那一支還會看圖片記錄 `+13h`／`+15h` 是否為零）|
| `[bp+0Ah]`／`[bp+0Ch]` | `[4670h]`／`[4672h]` | 圖片遠指標（剛由 entry 5 載進來的）|
| `[bp+6]`／`[bp+8]` | `[4980h]`／`[4982h]` | 目標緩衝遠指標 |

開頭兩道閘門是「來源與目標都不能是空指標」，兩者任一為 0 就跳到 `14CFh`
（收尾）。

**五張圖全部是 `(0, 0)`、旗標 `0`。** 所以位置不在引數裡——要嘛在目標緩衝
`DS:4980h` 指到的結構，要嘛圖片記錄自己帶。載入那一支（entry 5，`011Bh`）
的引數是 `(區塊編號, 0, 0, 目標遠指標)`，區塊編號依序是 4、5、6。

`[4937h]+67Ch` 是**逐格推進的計數**：第一張之後 `cmp ..., 1 / jbe` 跳過其餘，
第二張之後比 2，第三張之後比 3——它決定後三張畫不畫。

**還缺 `DS:4980h` 是什麼。** 讀完再接圖，位置不猜——猜錯的症狀是「圖出現在
錯的地方」，而那和「還沒接」在報表上分不出來。
