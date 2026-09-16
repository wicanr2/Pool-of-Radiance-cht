# Spec 116：估價與販賣寶石珠寶（overlay-21 entry 19）

狀態：READY（兩張價值表、賣價的比例、留著時建出來的物品欄位、
「留著」出不出得來的門檻都已讀出、實作並接上神殿的界面）；CONFORMED（武具店付款：
金幣等值、重鑄、角色先於 pool——dosgolem 兩筆收據逐欄相同）；
OPEN（留下來那件物品的其餘欄位；神殿仍只看金幣欄）。
日期：2026-09-05（2026-09-10：商店那一側的入口已經接上；2026-09-16：付款，#30）。

## 一支給兩個地方用

商店與神殿的選單裡都有 A）ppraise，**remake 兩邊都接上了**
（`cmd/pool-game/shop.go` 的 `offerShopAppraise` 與神殿那一側共用同一份規則）。說明書 p.34 講神殿時直接說「各種處理錢財
的指令都與在商店中的一樣，不再贅述」——原版也是同一支：overlay-04 與
overlay-06 的選單字串各自有 `Appraise`，實作都在 **overlay-21 entry 19
（`19F7h`）**。

寶石與珠寶不是物品，是角色記錄七欄貨幣裡的**兩個計數**：spec 040 的版面是
`+88h + 2 × 索引`，所以 `+90h` 是白金、`+92h` 是寶石、`+94h` 是珠寶。
這一支減的正是後兩欄、加的正是白金那一欄——三個欄位一起自洽。
一顆都沒有時印 `No gems or jewelry` 就返回。

## 寶石：1d100 查一格固定價

`1CB1h` 先把 `+92h` 減一，再擲 1d100：

| 1d100 | 金幣 |
|---|---:|
| 1–25 | 10 |
| 26–50 | 50 |
| 51–70 | 100 |
| 71–90 | 500 |
| 91–99 | 1000 |
| 100 | 5000 |

## 珠寶：1d100 選一段，段內再隨機

`1F40h` 先把 `+94h` 減一，再擲 1d100，然後 `Random(範圍) + 底價`
（Turbo Pascal 的 `Random(N)` 回 0..N−1）：

| 1d100 | 範圍 | 底價 | 值域 |
|---|---:|---:|---|
| 1–10 | 900 | 100 | 100–999 |
| 11–20 | 1000 | 200 | 200–1199 |
| 21–40 | 1500 | 300 | 300–1799 |
| 41–50 | 2500 | 500 | 500–2999 |
| 51–70 | 5000 | 1000 | 1000–5999 |
| 71–90 | 6000 | 2000 | 2000–7999 |
| 91–100 | 10000 | 2000 | 2000–11999 |

## 賣掉是換成白金付，不是折價

```
1e0e  問 `You can : ` / `Sell Keep`
1e16  按的不是 K → 賣掉
1f07  ax = 估價（金幣）
1f0d  cx = 5
1f10  ax ÷= 5
1f16  entry 9（0201h）把 ax 加進 +90h（白金）
```

估價的單位是**金幣**（畫面上印的是 ` gp.`），而錢加進去的是**白金**那一欄。
AD&D 一版 1 白金 ＝ 5 金，所以那個 `div 5` 是**幣別換算**——賣掉拿的是估價
的全額。除以 5 之後是整數除，所以不足 5 金的零頭會被無條件捨去。

`0201h` 不是無條件加：它先問 `0058h` 這個人還背得動多少（比記錄 `+102h` 的
重量與容量），塞不下的部分會進 `DS:6762h` 那個 32-bit 累加格。

按 K）eep 則是把它變成一件物品：配 3Fh（63）bytes 的記錄，
`+2Eh = 46h`、`+31h = 65h`、`+37h = 1`，估好的價值寫在 `+3Ah`。

**「留著」不是永遠出得來**：`1DA7h` 比記錄 `+C7h`（帶著幾件物品）是不是已經
到 10h（16）。滿了就只印 `Sell`，沒有 `Sell Keep`。

## 付款：金幣等值、重鑄、角色先於 pool（exact，2026-09-16，#30）

武具店的 B）uy（overlay-06 entry 4 `034Fh`）與神殿的付費（overlay-04，spec 018）用同三支
服務：

```
39d  價 = 物品 +3Ah
3af  有 = 00C9h:0057h(目前角色)                 ; overlay-19 entry 11（28A2h）
3ba  價 <= 有 → 0230h 收下物品（entry 2，spec 035）成功才 有 −= 價；00D9h:006Bh(有)
3e7  否則 pool = 00D9h:0075h(&6752h)           ; overlay-21 entry 17（00ACh）
       價 <= pool → 收下物品成功才 pool −= 價；00D9h:0070h(pool)
43f  都不夠 → "Not enough money."（cs:033Dh）
```

- **entry 11（`28A2h`）**：五種硬幣（`+88h` 銅、`+8Ah` 銀、`+8Ch` 琥珀金、`+8Eh` 金、
  `+90h` 白金）各乘 `DS:0D38h` 的換算值（1、10、100、200、1000 銅）加總，
  `(總銅 + 100) ÷ 200`——四捨五入到金幣。寶石、珠寶不算。
- **entry 15（`012Eh`，`gp`）**：清掉目前角色五種硬幣，`+90h 白金 = gp ÷ 5`、
  `+8Eh 金 = gp mod 5`。所以付完錢銀幣、銅幣、琥珀金都不見了，餘額全成白金加零頭的金。
- **entry 17／16（`00ACh`／`0183h`）**：pool 版（`DS:6752h` 起五個 longint）的同兩支。
- 角色出得起就角色出、餘額重鑄；不夠才整筆看 pool；不混付。

dosgolem 收據 `docs/audit/dosgolem-shop-payment.json`（`tools/dosgolem-shop-payment.py`，
走 `dosgolem-ref-shop` 那條路進武具店，`b,Return` 買 HAND AXE 1 金）：

| 錢包 | 買前等值 | 買後（原版） | remake |
|---|---|---|---|
| 白金 100 | 500 | 白金 99 金 4 | 同 |
| 銅 199 銀 19 金 30 白金 2 | (199+190+6000+2000+100)÷200 = 42 | 白金 8 金 1 | 同 |

實作 `treasure.GoldEquivalent`／`PoolGoldEquivalent`／`RemintCharacterMoney`／
`RemintPooledMoney`／`PayGold`；`cmd/pool-game/shop.go` 的 `buy` 照上面的順序
（`TestBuyingWithPlatinumOnly`、`TestBuyingFallsBackToThePartyPool`、
`TestShopPaymentMatchesTheDosgolemReceipt`）。神殿（`internal/temple`）仍只看金幣欄，
那是 spec 018 的既有簡化，換過來是同三支服務。

## 實作

`internal/treasure`：`GemValueBands`／`JewelryValueBands` 是兩張表，
`GemValue`／`JewelryValue` 查價，`SellPrice` 是那個五分之一。
計數存在存檔的 `Money[Gems]`／`Money[Jewelry]`，金幣是 `Money[Gold]`。

界面在 `cmd/pool-game/appraise.go`：神殿主選單的 A）ppraise 進兩層——先挑
寶石或珠寶，再對估好價的那一件選 S）ell／K）eep。留著時建出來的 63 bytes
記錄直接放進 `poolsave.Item.Raw`。

## OPEN

- 留下來的那件物品的其餘欄位（63 bytes 裡除了 `+2Eh`／`+31h`／`+37h`／`+3Ah`
  以外的）還沒對，所以那件東西目前只帶得動價值。
