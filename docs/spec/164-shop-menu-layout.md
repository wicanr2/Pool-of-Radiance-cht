# Spec 164：商店主選單的版面與店主肖像

狀態：CONFORMED（店主肖像的來源與像素、主選單的構圖、底列兩個選單字串、B）uy 之後
才出現貨品清單）；貨品清單那一頁本身的版面是 remake 的呈現（原版那一頁沒有 dosgolem 基準）。
日期：2026-09-27（#70）。

買賣、估價、公款與鑑定的規則在 spec 067 與 spec 116，這一份只管畫面。

## 原版長什麼樣（基準）

dosgolem 走進菲蘭武具店、答 `y` 之後那一幀：`workplace/dosgolem-ref-shop/79-y`
（`provenance.json`：generator `dosgolem`、revision `d351681b…`、`START.EXE`
`12811cbc…`）。同一張複製到 `docs/reference/original-dos/shop/79-y-weapon-shop.png`
（320×200，SHA-256 `ab41baf8…`）給測試用。

| 區域 | 原版（native） | 內容 |
|---|---|---|
| 左上那一框 | 內部 `(24,24)` 起 88×88 | 店主肖像 |
| 右上 | `NAME AC HP` 與名單 | 沒有座標時鐘那一行（進店前那一幀 `78-Up` 有 `8,11 E 00:15`） |
| 文字框 | 列 136..183 | 清空 |
| 底列 | 列 192..198，從 x=0 起 | `BUY VIEW POOL APPRAISE EXIT`，可按的字母色號 15、其餘 10 |

## 店主肖像從哪來（exact）

四家店都在印開場白之前寫 head 選擇子、再下 `PICTURE`（ECL3/block 0，
`workplace/ecl3-block0.json`，`ecl3.dax` SHA-256 `db58f0c6…`，code base `9900h`）：

| 店 | 位址 | 序列 |
|---|---|---|
| 雜貨店 | `A242h` | `SAVE 42 → @6DE1` ; `A248h PICTURE 9` |
| 珠寶店 | `A838h` | `SAVE 63 → @6DE1` ; `A83Eh PICTURE 34` |
| 武具店 | `A8BFh` | `SAVE 42 → @6DE1` ; `A8C5h PICTURE 9` |
| 銀器店 | `A950h` | `SAVE 63 → @6DE1` ; `A956h PICTURE 34` |

`6DE1h` 就是 `[4937h]+5C2h`（spec 117〈Rolf 的 head 區塊〉的換算），不是 `FFh` 時
`PICTURE n` 走 HEAD／BODY 那一條：`HEAD<區號>.DAX` 區塊 = `6DE1h` 的值、
`BODY<區號>.DAX` 區塊 = n。所以武具店是 `HEAD3/42` 疊 `BODY3/9`
（BODY 與 Rolf 同一塊，head 不同）。之後的 `GOSUB AE5Ah`（`HORIZONTAL MENU`）與
`CLEARMONSTERS → TREASURE → SAVE ×3 → COMBAT` 都不碰圖；離店後才 `PICTURE 255`。

像素對照：`HEAD3/42 + BODY3/9` 疊出來的 88×88 與基準那一框**逐格相同**；負對照是
換成 Rolf 的 head 8，對不上。

## 契約

1. 每一個 `0Eh PICTURE` 邊界都記下 `(當下的 6DE1h, 運算元)`（`applyCellECLResult` →
   `recordECLPicture`）。PICTURE 本身仍是純顯示的邊界（spec 082），這裡只記不畫。
2. 商店主選單（沒有開貨品清單、賣出頁或 T）ake）時，第一人稱那一框換成
   `loadNPCPortrait(區號, head, body)`；右邊只畫 `NAME AC HP` 與名單，不畫狀態列；
   文字框清空，有訊息（估價、公款、錢不夠、離店提問）才印在框裡。
3. 底列從 x=0 起畫 overlay-06 的兩個字串之一：公款沒錢是 `0487h`
   `Buy View Pool Appraise Exit`，有錢是 `0460h` `Buy View Take Pool Share Appraise Exit`；
   離店提問（`~Yes ~No`）時不畫。字母高亮與冒險指令列同一支（`drawCommandLabels`）。
   繁中標籤鍵名留在最前面（`B 購買`），同冒險與紮營那兩列。
4. 按 `B` 才開貨品清單；清單裡上下選、`ENTER` 買、`TAB` 換買家、`ESC` 回主選單。
   主選單上 `ENTER` 與方向鍵不買東西，`E` 與 `ESC` 離店。其餘鍵（V、P、S、T、G、J、TAB）
   照 spec 067／116。`A` 印出 G／J 的提示——原版的估價子選單是 remake 的 G／J 兩鍵。
5. `CAMP → ALTER → PICS` 關掉肖像、`6DE1h == FFh`（PIC 那一條，remake 還沒接）或
   區塊讀不到時退回第一人稱視野，商店照樣能用。肖像快取跟著換主題清掉。
6. 隊伍不只一人時，名單上目前的買家換強調色；一人時與原版同色。

## 驗收

- `TestShopkeeperPortraitMatchesTheDOSShopShot`：`HEAD3/42 + BODY3/9` 與基準逐格相同，
  Rolf 的 head 對不上。
- `TestAllFourShopkeepersCompose`：兩組 head／body 都疊得出 88×88。
- `TestNormalKeysBuyAndEquipFromTheWeaponShop`：從標題正常按鍵走進武具店時記下的是
  `(42, 9)`、停在主選單；按 `B` 開清單、買、`ESC` 回主選單、再 `ESC` 離店。
- `TestShopMenuKeysOpenAndCloseTheBuyList`、`TestShopMenuCommandsFollowThePool`、
  `TestShopkeeperPortraitComesFromTheLastPicture`（快取、PICS 關、PIC 那一條）。
- 發行包對拍 `shop` 那一張（`tools/appimage-dos-parity.sh`），數字記在 commit message。

## 不做

- `PICTURE` 在其他場合（進店前的開場白、神殿、城區事件）的顯示。那些畫面各有自己的
  對拍項，這一份只接商店主選單。
- PIC 那一條（`PIC<區號>.DAX` 的動畫容器，spec 117〈PIC 容器的版面〉）。
- 貨品清單那一頁的原版版面；需要 dosgolem 按 `b` 之後的基準才能對。
