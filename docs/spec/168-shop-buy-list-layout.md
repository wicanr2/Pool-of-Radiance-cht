# Spec 168：商店貨品清單那一頁

狀態：CONFORMED（版面、清單順序、一打開的反白、翻頁與底列；dosgolem 基準與發行包對拍
`shop-buy`）。反白在最後一項再往下、NEXT 之後不足一頁時的起點是 remake 的做法（見〈不知道的〉）。
日期：2026-09-29（#124）。接 spec 164：主選單照原版，按 `B` 之後的這一頁由本份負責。

買賣、估價、公款的規則在 spec 067 與 spec 116，這一份只管畫面與清單裡的鍵。

## 原版長什麼樣（基準）

dosgolem 走 `workplace/dosgolem-ref-shop` 那條鍵序（與 spec 164 的主選單同一條，
結尾多一個 `b`）。那一條產出的 `79-y`（主選單）雜湊仍是 `02d78f41`，所以主選單那一張
的基準沒有變；多出來的 `80-b` 就是這一頁：

- `provenance.json`：generator `dosgolem`、revision `2ab6f651…`（`fix/pool-wipe-anykey`）、
  `START.EXE` `12811cbc…`、81 幀。
- `80-b.idx` SHA-256 `4d5cbdcf50a8bbfc…`。同一張 PNG 複製到
  `docs/reference/original-dos/shop/80-b-buy-list.png`（SHA-256 `4ddf2f10…`）。
- 人類牧師那條路（`workplace/dosgolem-ref-spells` 的 `76-b`）也是 `4d5cbdcf`：這一頁不畫隊伍，
  誰進店都一樣。

另外拍了一條探索用的鍵序（不進對拍）：同一條路之後 `b,End,End,Home,n,n,n,e,b`，
量翻頁與重開（`workplace/dosgolem-probe-shopbuy`，89 幀）。

| 區域 | 原版（native） | 內容 |
|---|---|---|
| 外框 | 整頁那一圈（spec 123），框內清空 | 沒有第一人稱框、沒有名單 |
| 標題 | 列 8..14，第 17..20 欄 | `SHOP`，色號 15 |
| 清單 | 列 32..182，行距 8，一頁 19 行 | 名稱從第 1 欄（字身 x=9）、價格靠右到第 30 欄（x=247） |
| 反白 | 那一行名稱與價格都是色號 15 | 其餘色號 10 |
| 底列 | 列 192..198，從 x=0 起 | `ITEMS: BUY NEXT EXIT`；`ITEMS:` 色號 13，B／N／E 色號 15，其餘 10 |

## 清單順序（exact，兩條路逐行讀出）

原版的清單是 ITEM 記錄的**倒序**。武具店的存貨是 `ITEM3.DAX` block `35h` 的 57 筆，
檔案第一筆是 `Shield`、最後一筆是 `Battle Axe`；原版第一頁是

`BATTLE AXE, HAND AXE, BARDICHE, BEC DE CORBIN, BILL-GUISARME, BO STICK, CLUB, DAGGER,
4 DARTS, FAUCHARD, FAUCHARD-FORK, FLAIL, MILITARY FORK, GLAIVE, GLAIVE-GUISARME,
GUISARME, GUISARME-VOULGE, HALBERD, LUCERN HAMMER`，

第二頁從 `HAMMER` 起、第三頁最後一行是 `SHIELD`（`84-n`／`85-n`）。57 = 3 × 19，三頁剛好。

名稱走組名稱那一支（overlay-25 entry 1，`internal/treasure.ItemName`），所以帶數量的記錄是
複數：檔案的 `4 Dart` 在清單上是 `4 DARTS`，`20 Quarrel(s)` 是 `20 QUARRELS`。

## 鍵與反白（strong inference，量自探索那一條）

| 鍵 | 原版 | 量到的 |
|---|---|---|
| 開清單 | 反白在**第二項** | 第一次開：第 0 行起、反白第 1 行（`80-b`）；翻到最後一頁、`e` 離開再 `b`：從 `HAND AXE` 起、反白第 0 行（`88-b`）——反白回到第二項，起點只移到看得見它 |
| `End`／`Home` | 反白往下／往上一行 | `81-End` 第 2 行、`82-End` 第 3 行、`83-Home` 回第 2 行 |
| `N`）ext | 整頁往後，頁內那一行不變 | `84-n` 第二頁、反白第 2 行（`JO STICK`）；`85-n` 第三頁第 2 行（`COMPOSITE LONG BOW`） |
| 最後一頁 | 底列沒有 `NEXT` | `85-n` 底列 `ITEMS: BUY PREV EXIT`，再按 `n` 畫面不動（`86-n`） |
| 中間頁 | 兩個都有 | `84-n` 底列 `ITEMS: BUY NEXT PREV EXIT` |
| `E`）xit | 回主選單 | `87-e` 與 `79-y` 同一個雜湊 |
| `B`）uy | 買反白那一件 | spells 那條路 `b,b` 買到 HAND AXE（spec 116 的付款收據） |

原版的上下鍵不作用（spec 133 那一族），remake 另外接。

## 契約

1. `enterShop` 把讀到的記錄整串倒過來（`shopStock`）。不只一個 block 的店目前沒有；
   照同一條倒過來是推論。
2. 開清單（主選單的 `B`）時反白放在第二項（只有一項就是第一項），起點 `top` 只在反白
   不在這一頁時才移（`keepCursorVisible`）。`top` 跟著這一趟進店留著。
3. 清單裡：`B`／`ENTER` 買、`N`／`PgDn` 下一頁、`P`／`PgUp` 上一頁、`End`／`Down` 下一行、
   `Home`／`Up` 上一行、`E`／`ESC` 回主選單、`TAB` 換買家。`N` 只在 `top + 19 < 件數`
   時作用，`P` 只在 `top > 0` 時作用；翻頁時 `top` 與反白一起移 19 行。
4. 版面（邏輯座標是 native 的兩倍，基線換算同 spec 134）：框內 `(16,16)..(624,366)` 清空；
   標題以 x=304 置中、基線 30；清單第一行基線 78、行距 16、名稱左緣 18、價格右緣 496；
   反白那一行用強調色。
5. 底列從 x=0 起：前綴 `Items: `（繁中 `物品：`，色同紮營的前綴，`commandPrefixInk`），
   接 `Buy [Next] [Prev] Exit`，字母高亮走 `drawCommandLabels`。有訊息（買到、錢不夠、
   拿不動）時底列換成那一句。
6. 隊伍不只一人時，標題與清單之間（native 24..30 那一列，原版留白）印出目前的買家與他的
   金幣等值；一人時與原版一樣留白。原版的買家是主選單上選定的人，不寫在這一頁。
7. 截圖用的畫面識別字是 `shop-buy`（`screen_state.go`）。

## 價格欄

**記錄價格為 0 的貨品上架時改成 1 金，清單印 1、買也收 1（exact）。**

武具店有五筆：`4 DARTS`、`2 JAVELINS`、`20 QUARRELS`、`10 ARROWS`、`SLING`。
原版清單上都印 `1`（`80-b` 第 8 行、`84-n` 第 1／8 行、`85-n` 第 6／8 行逐格讀出）。

位元組（`GAME.OVR` SHA-256 `bc4e3c32…`，overlay-06 `workplace/ovr/overlay-06.bin`
SHA-256 `2db20078…`，overlay 內 offset，`objdump -m i8086` 反組譯）：

```
0030h  清單那一支：les ax,[676Eh]   ; 存貨鏈的頭
0064h  C4 7E F8          les di,[bp-8]            ; 這一筆
0067h  26 83 7D 3A 00    cmp word es:[di+3Ah],0
006Ch  75 09             jne 0077h
0071h  26 C7 45 3A 01 00 mov word es:[di+3Ah],1   ; 寫回記憶體裡的那一筆
007Ah  26 8B 45 3A       mov ax,es:[di+3Ah]       ; 印價格
```

B）uy（entry 4 `034Fh`）一進來就 `0383h call 0030h` 印清單，`039Dh` 付款讀的是同一筆的
`+3Ah`，所以收的就是改過的 1。全 38 個 overlay 掃 `26 C7/89 .. 3A` 寫 `+3Ah` 的指令，
overlay-06 只有 `0071h` 這一條（其餘在 overlay-21，是估價與戰利品那一路）。

runtime 收據（dosgolem `2ab6f651`，`START.EXE` `12811cbc…`，`workplace/dosgolem-probe-price`）：
ref-shop 那條鍵序進店，`v` 看人物資料頁 `GOLD 120`（`80-v`）；`b`、`End` ×7 反白 `4 DARTS`、
`b` 買、`e`、`v`：`PLATINUM 23 GOLD 4`（`92-v`，119 金）；再買一次：`PLATINUM 23 GOLD 3`
（`104-v`，118 金）。負重 27 → 47 → 66，兩次都真的收下了。

remake：`shopStock` 上架時把價格 0 的記錄寫成 1（`TreasureItemRecord.SetPrice`），買下的那一件
也帶著 1——與原版寫回記憶體裡那一筆相同，之後賣出的出價照 `+3Ah` 算。

## 驗收

- `TestShopListFirstPageMatchesTheDOSShot`：第一頁 19 行的名稱與基準逐行相同，反白在
  `HAND AXE`，底列 `Buy Next Exit`。
- `TestShopListPagesLikeTheOriginal`：End／Home、NEXT 保留頁內行、中間頁與最後一頁的底列、
  最後一頁 NEXT 不作用、離開再開回到 `HAND AXE`。
- `TestShopListBBuysTheHighlightedItem`：`B`、`B` 買到手斧。
- `TestShopListChargesOneForZeroPricedAmmunition`：從 Update() 送 `b`、`End` ×7、`b`、`e`
  兩輪，錢包 120 → 白金 23 金 4 → 白金 23 金 3，與 dosgolem 收據相同。
- `TestShopStockOnlyRaisesZeroPrices`：只有那五筆從 0 改成 1，其餘價格不動。
- 發行包對拍 `shop-buy`（`tools/appimage-dos-parity.sh`，基準 `ref-shop` 的 `4d5cbdcf`），
  數字在 `docs/audit/dos-parity-sample.*`。旅店那一段買手斧改成與原版同一組鍵
  `b,b,e,e`。

## 不知道的

- 反白在最後一項再往下：原版沒拍；remake 繞回第一項（沿用 #70 之前的行為，主線測試靠它
  走到指定的一件）。
- 起點不在 19 的倍數時按 NEXT（例如重開之後 `top = 1`）：remake 讓 `top` 加 19、可以不足一頁；
  原版沒拍。
