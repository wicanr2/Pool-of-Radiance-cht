# Spec 067：商店服務邊界與進貨清單

狀態：CONFORMED（服務邊界的判別、四家店的存貨來源、價格欄、購買與付款、
賣出的出價與收款、進店清空公款與離店的「落下錢」提問、物品選單的 I）d 鑑定——
付款與公款的前後值有 dosgolem 收據）；DRAFT（原版商店選單的版面；鑑定揭露藏字那一支
只有位元組與內嵌名稱的正對照，沒有原版實跑）。
多幣別付款由 `treasure.PayInCoins` 接上（#30）。
日期：2026-09-03（2026-09-25：賣出，#60；2026-09-26：公款 #67、鑑定 #68）。

## 商店與戰利品是同一條邊界

四家店與墓園戰利品都走 `CLEARMONSTERS → TREASURE → SAVE… → COMBAT`。
把 `ECL3/block0` 裡每一條這樣的序列列出來，差別一眼可見：

| 位址 | 序列 |
|---|---|
| `A29Dh` 雜貨店 | `TREASURE 0,0,0,0,0,0,0,54` ＋ `6E6Ch=1` `6EF6h=1` `6E6Dh=16` |
| `A892h` 珠寶店 | 同上，item block `52` |
| `A919h` 武具店 | 同上，item block `53` |
| `A9AAh` 銀器店 | 同上，item block `55` |
| `A780h` 墓園戰利品（spec 032）| `TREASURE` 帶七種貨幣的數量，item block `51`，**沒有那三個旗標** |

所以判別要看那三個旗標，**不能看 item block 編號**：編號只說「存貨在哪一格」，
之後若有新的戰利品用到相鄰的 block，用編號判會把它當成商店。

沒有這個判別的後果不是當掉：`enterTreasure` 會照收，玩家走進武具店就拿到
57 件免費裝備——而畫面上那看起來像一堆戰利品，不像分派錯了。

## 存貨就是 `ITEM3.DAX` 的四個 block

`TREASURE` 請求的第八欄是 item block（engine 的 `TreasureRequest.ItemBlock`），
格式與墓園戰利品完全相同（spec 033 的 63-byte 記錄）。

| block | 店 | 件數 | 樣本 |
|---|---|---:|---|
| `34h` | 珠寶店 | 11 | 黃銅龍雕像 75、鑽石項鍊 50000 |
| `35h` | 武具店 | 57 | 長劍 15、板甲 400、複合長弓 100 |
| `36h` | 雜貨店 | 7 | 聖水 25、蘇妮的銀製聖徽 50、油瓶 1 |
| `37h` | 銀器店 | 13 | 銀長劍 150、精製複合長弓 25000 |

## 價格在記錄的 `+3Ah`（word，金幣）

四個獨立的對照都與 AD&D 玩家手冊相同：長劍 15、板甲 400、匕首 2、鏈甲 75。
同一欄在 block `35h` 的其他記錄也都對得上，因此欄位位置是 exact。

**箭、弩矢、標槍、投石索的價格是 0。** 那是原始資料就寫 0，不是讀錯欄位——
同一個欄位在同一個 block 的其他 55 筆全部對得上原書價目。這一點照實記下，
不替它補一個「應該是多少」。

## 跨規格的一致性

武具店的存貨同時替 spec 065 做了獨立驗證：`Long Sword` 記錄的 `+2Eh` 是 `24h`，
而物品型別表第 `24h` 筆是 1d8／1d12——AD&D 的長劍；`Two-Handed Sword` 是 `26h`
對 1d10／3d6；`Short Sword` 是 `25h` 對 1d6／1d8。名稱、型別索引與骰子三者
在四十幾筆上同時成立，不可能是巧合。

## 購買

付款照原版的五種硬幣換算：先算買家的金幣等值，夠就從他身上扣、餘額重鑄成
白金＋金，不夠才看隊伍公款，兩邊不混付（spec 116〈付款〉，`treasure.PayGold`）。

買入沿用 spec 035 的 `CanReceiveItem`：16 格上限與力量負重同樣適用，拿不動就
不賣、也不扣錢。買來的物品 `+34h` 是 0，跟撿到的一樣要自己裝上（spec 065）。
錢與物品同時寫進隊伍與角色庫，兩邊不會分岔。

離開（`ESC`）讓 ECL 從 `COMBAT` 邊界之後續行，與神殿離開（spec 017）走同一條路。

## 賣出

輸入：`game.ovr` 的 overlay-06（`2db20078…`）、overlay-19（`4694cb51…`）、
overlay-21（`8324f587…`）、overlay-22（`967065cc…`）、overlay-25（`9fede24b…`），
SHA-256 與 `docs/audit/dos-ovr-manifest.json` 的 `code_sha256` 相同；位址是各 overlay
的檔內位移，以 `objdump -D -b binary -m i8086`（`coab-go-test:20260729`）讀，far call 以
spec 109 的 stub segment 換算。

### 選單怎麼接（exact）

商店選單本身沒有 Sell。賣東西是 V）iew → I）tems → 選物品 → S）ell：

```
overlay-06 0531  C6 06 54 49 01        mov [4954h], 1        ; 進店：物品選單的「商店模式」
overlay-06 0619  3C 56 75 0D ...       'V' → 9A 39 00 C9 00   ; overlay-19 entry 5（0AB6h）人物資料頁
overlay-19 0CA9  3C 49 ...             'I' → E8 45 02         ; entry 6（0EFBh）物品選單
overlay-19 10EA  80 3E 54 49 01 75 28  [4954h] == 1 才把 " Sell"（0EABh）接進選項
overlay-19 1119  80 3E 54 49 01 75 28  同一條件接 " Id"（0EB1h）
overlay-19 1432  3C 53 75 20           'S' → E8 32 F9（entry 20 0D72h 放手檢查）
                                        → 過了才 E8 9B 08（entry 16 1CE9h 出價與付錢）
```

選項裡的 Sell 另有一個角色條件（`10C9h..10E8h`）：記錄 `+84h < 80h`、或 `+10Dh == 0`、
或 `+10Ch == 1` 才放。玩家建的角色 `+84h` 是 0，一律有 Sell；只有 ADD NPC 帶進來、
`+84h` 位元 7 立著（有士氣判定）而且 `+10Dh` 非 0 的 NPC 看不到它（Trade 同一條件，
`0FF6h..1017h`；Drop 在 `1046h` 一律接，spec 144）。
remake 對 `+84h`／`+10Dh` 讀 NPC 的原始記錄；`+10Ch` 取 `Character.Status`——兩者是否
逐值同義未另證，這一格是 strong inference。

原版實跑的畫面（dosgolem，見下方收據）：物品選單底列是
`READY TRADE DROP HALVE JOIN SELL ID EXIT`，按 S 之後印
`I'll give you N gold pieces for your <物品>`，底列 `Is It a Deal? Yes No`。

### 放手檢查：哪些東西不收（exact）

entry 20（`0D72h`）是 Trade、Drop、Sell 共用的「放不放得了手」：

1. `0D7Fh` `26 80 7D 34 00`：`+34h` 非 0（穿戴中）→ 印 `Must be unreadied`（`0D22h`），不賣。
2. `0DA3h` `9A 3E 00 E2 00`：overlay-22 entry 6（`31F6h`）認卷軸——物品型別表
   `DS:54E0h + 型別×16` 的第 0 格大於 0Ah 且小於 0Eh。是卷軸而且 `+3Ch`／`+3Dh`／`+3Eh`
   任一格大於 7Fh（有人正要從它抄法術）→ 印 `<名字> was going to scribe from that scroll`
   ／`is it Okay to lose it?`，Y 才放手。
3. 其餘一律放手。

**沒有別的拒收條件**：不看類別、不看鑑定與否、不看詛咒。被詛咒的東西卸不下來，
所以它是因為第 1 條賣不掉，不是有一條詛咒檢查。值 0 的東西照樣出價 0 並收走
（HAND AXE 值 1 → 出價 0，原版實跑確認）。

### 出價（exact）

entry 16 開頭（`1CF0h..1D52h`），`[bp+6]` 是物品記錄：

```
1CF0  31 C0 89 46 FE                 價 = 0
1CF8  26 83 7D 3A 00 76 11           +3Ah > 0 →
1D02  26 8B 45 3A 31 D2 B9 02 00 F7 F1   價 = +3Ah div 2
1D13  26 80 7D 39 01 76 3B           +39h（數量）> 1 →
1D1D  26 80 7D 2E 49 75 0A           +2Eh ≠ 49h → 1D2E
1D27  26 80 7D 2E 1C 74 18           +2Eh == 1Ch → 1D46（不除）
1D2E  … F7 66 FE 31 D2 B9 14 00 F7 F1  價 = (數量 × 價 的低 16 位) div 20
```

`1D46h` 那一支（只乘不除）要 `+2Eh` 同時等於 49h 與 1Ch 才走得到——同一格不可能，
所以**數量大於 1 一律除以 20**。49h 是箭、1Ch 是弩矢（`ITEM3.DAX/35h` 的
`10 Arrow(s)`、`20 Quarrel(s)`），看得出原意是「箭與弩矢不除」，但位元組上那條
比較寫成了兩個都要成立。remake 照位元組：`treasure.SellOffer`。
`mul` 之後 `xor dx,dx` 丟掉高位，所以乘積超過 65535 會繞回，也照做。

### 錢怎麼給（exact）

按 Y 之後（`1DE6h..1EACh`）：

1. 印 `Sold!`，overlay-25 entry 17（`156Ah`）把物品從 `+C8h` 串列摘掉並釋放 3Fh bytes。
   這一支**不動 `+102h`**；負重要等選單迴圈在 `146Fh` 呼叫 overlay-25 entry 7 才重算。
2. `白金 = 價 div 5`、`金 = 價 mod 5`（`1E0Dh..1E25h`）。估價的單位是金幣，1 白金 = 5 金。
3. overlay-21 entry 4（`0058h`）：`+102h + (白金 + 金)` 大於負重上限（entry 0：
   overlay-25 entry 15 的力量調整 + `5DCh` = 1500）就回「超重」，並把
   `上限 − +102h` 寫進可容納量。**這時的 `+102h` 還含著剛賣掉那件的重量**（見第 1 條）。
4. 沒超重：白金加進 `+90h`、金加進 `+8Eh`（`1E99h..1EACh`）。
5. 超重：印 `Overloaded.  Money will be put in pool.`；可容納量大於白金就白金全給角色，
   否則角色拿可容納量、其餘白金加進公款白金 `DS:6762h`（32 位元）；金一律加進 `+8Eh`。

加法都是 16 位元 `add`，溢位繞回。**不重鑄**：賣出只加兩欄，身上原有的銅、銀、
琥珀金不動——與購買（spec 116）付完錢把五種硬幣重鑄成白金＋金不同。

實作：`internal/treasure/sell.go`（`SellOffer`、`SellNeedsUnready`、
`SellNeedsScribeConfirm`、`SellItem`）。負重沿用 `+102h` 的算式（物品重量 × 數量＋
七欄錢的枚數，spec 079），在摘掉物品之前算。

### 界面

商店選單按 `V` 直接列出目前這個人（`TAB` 換人）的物品——原版先開人物資料頁再按 I，
remake 在商店裡沒有那一頁，所以少一步；之後的鍵與原版相同：上下選、`S` 賣、
出價後 `Y` 成交／`N` 不賣，`ESC` 回到架上。字串走 locale（`ui.shopSell*`），繁中與英文
各一份。商店標頭的錢改印金幣等值（entry 11），因為付款與賣出都把錢放在白金那一欄，
只印金幣欄會看起來錢沒動。

### 原版對照

dosgolem 收據 `docs/audit/dosgolem-shop-sell.json`（`tools/dosgolem-shop-sell.py`；
與 spec 116 付款收據同一個前置角色、同一條進店鍵序）：

| 情境 | 物品（`+3Ah`） | 原版出價 | `+102h` | 錢包 賣前 → 賣後 | 公款白金 | remake |
|---|---|---:|---:|---|---|---|
| partisan | Partisan（10） | 5 | 178 | 白金 98 → 99 | 0 → 0 | 同 |
| long-sword | Long Sword（15） | 7 | 157 | 白金 97 → 白金 98 金 2 | 0 → 0 | 同 |
| overload | Plate Mail（400） | 200 | 1700 | 白金 1250 → 1250 | 170 → 210 | 同 |

overload 那一筆是 P）ool 全部進公款、公款付錢買板甲、S）hare 平分把角色塞到負重
上限 1700（力量 14）之後再賣：可容納量 0，40 白金全部進公款。三筆的 `+102h` 都等於
「物品重量＋錢的枚數」而且**含著要賣的那一件**，與〈錢怎麼給〉第 3 條一致。
出價由各情境 `offer_frame` 那一幀的畫面讀出（`I'll give you N gold pieces`），
另外 dosgolem 在 PARTISAN 之前實跑過一次 HAND AXE（值 1）：出價 0、錢包不動、物品收走。

`TestShopSaleMatchesTheDosgolemReceipt` 用同名存貨、同一個錢包從 `Update()` 按 V S Y，
出價、五個錢欄與公款白金逐欄比對。

## 公款

輸入同〈賣出〉：overlay-06（`2db20078…`）、overlay-21（`8324f587…`）；START.EXE
（`12811cbc…`）的 RTL 段 `05BBh` 以 MZ header `3B0h` 換算檔內位移。

### 進店清成 0（exact）

overlay-06 entry 1（`052Ah`）是商店本體，一進來先做：

```
0531  C6 06 54 49 01              mov [4954h], 1           ; 物品選單的商店模式
0548  BF 52 67 1E 57              push ds:6752h            ; 公款七欄（spec 040）
054D  B8 1C 00 50 B0 00 50        push 1Ch, push 0
0554  9A B5 16 BB 05              call 05BBh:16B5h
```

`05BBh:16B5h`（START.EXE 檔內 `7615h`）是 `8B DC 36 C4 7F 08 36 8B 4F 06 36 8A 47 04
FC F3 AA CA 08 00`：`les di,[bx+8]; mov cx,[bx+6]; mov al,[bx+4]; cld; rep stosb; retf 8`
——Turbo Pascal 的 `FillChar(dest, count, value)`。所以進店把 `DS:6752h` 起 28 bytes
（七個 longint）全部寫 0，**不發還給任何人**。同一個 `FillChar(DS:6752h, 1Ch, 0)` 的形狀
另外出現在 overlay-03 `1357h` 與 overlay-04 `0D02h`，那兩處不在本規格範圍。

remake：`enterShop` 把 `State.PooledMoney` 清成 0。

### 離店時公款有錢就問（exact）

商店選單迴圈（`05A8h..077Ah`）每一輪先呼叫 overlay-21 entry 14（`0F2Bh`）：七欄公款
（`[6752h+4i]` 的高低字 `or`）有一欄非 0 就把旗標設 1。旗標決定選單字串：
`0460h` `Buy View Take Pool Share Appraise Exit`（有錢）或 `0487h`
`Buy View Pool Appraise Exit`（沒錢）。按 E（`0665h`）時：

```
0676  9A 66 00 D9 00      overlay-21 entry 14 → 旗標
067B  80 7E D1 00 75 03   旗標 == 0 → 0724h：離開旗標 = 1（直接離店）
0684  BF A3 04            選項 `~Yes ~No`
06AF  BF AC 04            As you leave the shopkeeper says, "Excuse me but you have left some money here."
06D5  BF FF 04            Do you want to go back and get your money?
06FD  9A 89 00 45 00      overlay-07 entry 21：選單，回傳選項索引（spec 086）
0705  80 7E CF 01 75 06   索引 == 1（No）→ 離開旗標 = 1
0711..071D                否則（Yes）清掉文字區（overlay-37 entry 14），回到商店選單
```

離店那一條**不動公款**：錢留在 `DS:6752h`，直到下一次進店被 `0548h` 清掉。
S）hare（`0648h` → overlay-21 entry 7）、T）ake（`062Ah` → entry 8）只在有錢的那一版
選單上，與戰利品同一支（spec 040）。

remake：`ESC` 時 `hasPooledMoney()` 為真就印這兩句、等 `Y`／`N`；`N` 離店、公款不動，
`Y` 回到商店。`ESC` 在這個提問上不作用——overlay-07 entry 21 對 ESC 的回傳沒有讀，
不猜。商店選單接上 `S` 平分公款（`ShareMoney`），讓「回去拿錢」有路可走；
T）ake 挑幣別與數量那一段商店裡還沒有（戰利品畫面有）。

### 原版對照（公款）

dosgolem 收據 `docs/audit/dosgolem-shop-pool-identify.json`
（`tools/dosgolem-shop-pool-identify.py`，前置角色與進店鍵序同〈賣出〉）：

- `leave-yes`：P 之後公款白金 100，按 E 印出 `AS YOU LEAVE THE SHOPKEEPER SAYS "EXCUSE ME
  BUT YOU HAVE LEFT SOME MONEY HERE." DO YOU WANT TO GO BACK AND GET YOUR MONEY?`，底列
  `YES NO`；按 Y 回到 `BUY VIEW TAKE POOL SHARE APPRAISE EXIT`，公款不動。
- `identify-pool` 的後段：公款白金 58 時 E → N，畫面回到城區 (8,11)，公款**仍是 58**；
  往西退一格再走回來按 y 進店，那一步之後公款是 **0**。離店不清、進店才清，兩半都看到了。

同一份收據裡，P 之後直接按 S（`leave-yes` 的 `p,s` 與 `y,s`）公款沒有動；〈賣出〉的
overload 情境是買過一件之後按 S，錢有分下去。差別的成因沒有追，remake 的 S 照
overlay-21 entry 7 的位元組做（spec 040）。

## 鑑定

輸入：overlay-19（`4694cb51…`）、overlay-25（`9fede24b…`）、START.EXE（`12811cbc…`）。

### 選項與入口（exact）

物品選單（overlay-19 entry 6）`1119h`：`[4954h] == 1` 就把 ` Id`（`0EB1h`）接進選項，
**沒有**賣出那一條角色條件（`10C9h..10E8h` 只管 Sell）。按 I（`1456h`）呼叫 entry 17
（`1F52h`），傳入選中的物品與一個「清單變了」旗標。

### 流程（exact）

```
1F77  9A 25 00 0A 01      overlay-25 entry 1：依記錄重組名稱，寫回 +0
1F94  BF C8 1E            For 200 gold pieces I'll identify your <名稱>
1FB2  BF F0 1E            Is It a Deal?
1FC5  9A 3E 00 1D 01      overlay-26 entry 6 取鍵；1FCA `3C 59`：不是 'Y' 就結束，不收錢
1FDA  E8 C5 08            entry 11（28A2h）角色的金幣等值
1FE3..1FF0                ≥ C8h（200）→ 1FF2：付了；1FF6 減 200；2002 overlay-21 entry 15 重鑄
2009  BF 52 67 …          否則公款（overlay-21 entry 17）≥ 200 → 付了；2038 entry 16 重鑄
203F  BF FF 1E            兩邊都不夠：Not Enough Money，結束
205F  26 80 7D 35 00      付了錢：+35h == 0 →
207E  BF 10 1F              I can't tell anything new about your <名稱>（錢不退）
2099  26 C6 45 35 00      否則 +35h = 0
20BB  9A 25 00 0A 01        重組名稱
20D8  BF 36 1F              It looks like some sort of <名稱>
20F1  26 C6 05 01           「清單變了」= 1
```

付款與購買同一條（spec 116〈付款〉，`treasure.PayGold`）：先看角色、不夠才看公款、
不混付，付完五種硬幣重鑄成白金＋金。**鑑定只改 `+35h` 與名稱**，記錄其餘欄位不動。

重鑄那兩支（overlay-21 entry 15／16，`retf 2`）只收一個 word，而 `1FFFh`／`2035h` 推的是
32 位元餘額的低位字；餘額超過 65535 金會繞回。購買（overlay-06 `0435h`）同一個形狀。
`PayGold` 目前沒有照這個截斷，影響購買與鑑定兩邊，另案處理。

### 名稱怎麼組（exact）

overlay-25 entry 1（`0441h..0753h`）。`+2Fh`／`+30h`／`+31h` 是三段字詞編號，查
`DS:10BBh + 編號 × 15h`（`0610h..061Dh`；START.EXE 檔內 `10BBh + 30640`，256 格 × 21 bytes，
例：`24h` Long Sword、`A2h` +1、`A7h` of、`B1h` Silver）：

1. 數量 `+39h` 大於 0：先放 `Str(數量) + " "`（`0527h..056Bh`）。
2. 遮罩（`0570h..05C6h`）：第 i 段（i = 1..3，對 `+2Eh+i`）非 0，而且
   `(+35h >> (3−i)) & 1 == 0` 才算看得見——`+35h` 位元 2 藏 `+2Fh`、位元 1 藏 `+30h`、
   位元 0 藏 `+31h`。
3. 由 i = 3 往 1 排看得見的段，每段後面接 `" "`；數量 ≥ 2 而且還沒接過時，下列條件
   成立的那一段改接 `"s "`（`066Bh..06D4h`）：只有這一段看得見；i = 1、遮罩 > 4、
   型別 `+2Eh` ≠ 56h；i = 2 而 `+2Fh` 看不見；i = 3 而型別 == 56h；型別是 49h 或 1Ch
   而 `+31h` ≠ B1h。
4. 每寫一次都截到 40 字（RTL `064Eh` 的長度上限 28h）。

另有兩個呈現用的前綴本規格不做：參數 `[bp+8]` 非 0 時的 ` Yes  `／` No   `（穿戴欄），
以及隊伍有人帶效果 5（`21DCh`）而物品 `+32h`／`+33h` > 0 或 `+36h` ≠ 0 時的 `* `。
鑑定呼叫時前者傳 0；後者 remake 還沒有對應的效果，一律當沒有。

正對照：`ITEM*.DAX` 裡數量為 0 的記錄共 311 筆，內嵌名稱與重組結果除尾端空白外逐字
相同（85 筆有藏字），例外是四筆手寫的名稱（兩筆 `…+3 vs. Undead`、一筆卷軸寫法術名、
一筆項鍊前導空白）。數量大於 0 的記錄內嵌的是手寫的 `10 Arrow(s)`、`3 Potion`，執行時
顯示的是這一支組出的 `10 Arrows `——`Boots` 也照樣接 `s`，得到 `Bootss`。

### remake

`treasure.ItemName`、`treasure.IdentifyItem`；字詞表 `gamepack.ReadDOSItemNameTable`。
商店的物品頁（`V`）按 `I` 印出價、`Y` 付錢鑑定、`N`／`ESC` 不要。鑑定後記錄 `+35h` 是 0，
名稱照原版帶尾端空白寫回記錄與 `Item.Name`（存檔要求兩者一致）；畫面上印的時候去掉。

### 原版對照（鑑定）

同一份 dosgolem 收據，買一把 PARTISAN（`+35h` 是 0）之後 V I I Y：

| 情境 | 按 Y 前 | 按 Y 後 | remake |
|---|---|---|---|
| identify-paid | 錢包白金 98 | 白金 58 | 同 |
| identify-short | 錢包白金 2、公款 0 | 不動 | 同 |
| identify-pool | 錢包 0、公款白金 98 | 公款白金 58 | 同 |

提示畫面是 `FOR 200 GOLD PIECES I'LL IDENTIFY YOUR PARTISAN` ／ `IS IT A DEAL? YES NO`。
付完錢的那一句（`I can't tell anything new…` 或 `Not Enough Money`）在按 Y 之後那一幀
已經換回物品選單，收據裡沒有拍到。**揭露藏字**（`+35h` 非 0 的物品）原版沒有實跑：
武具店的存貨 `+35h` 全是 0，要有一件帶藏字的物品得先從戰利品拿到。

`TestShopIdentifyMatchesTheDosgolemReceipt` 用同名存貨、同一個錢包與公款從 `Update()`
按 V I Y，五個錢欄與公款逐欄比對。

## 店在哪一格

四家店都由城區的地形索引分派（spec 102）：地形碼低七位是 `9B4Ch` 那張
`ON GOTO` 的索引，武具店是索引 22（`A8BBh`「THE SHOP SPECIALIZES IN ARMS
AND ARMOR」，`A919h` 就在它的 YES 分支底下）。geo3/0 裡索引 22 的格子有五格：
(13,8)、(8,11)、(11,12)、(8,13)、(9,13)。

## 驗收

- `TestRealArmouryBytesEnterTheShopService`：拿真的 ECL bytes 從 `A919h` 跑到
  服務邊界，確認判別認出商店而不是戰利品，且 `ITEM3.DAX/35h` 的存貨載進來。
- `TestArmouryStockCarriesTheOriginalPrices`：六件商品的價格與原書相同。
- 購買、金幣不足、買來未裝備、旗標判別各有一條測試。
- `TestNormalKeysBuyAndEquipFromTheWeaponShop`：**只用按鍵**走完整條鏈——
  建角拿到金幣 → 從開場結束的 (0,4) 走到 (8,11) 的武具店 → 買盾 →
  按 I 裝上 → AC 由 10 變 9。前面那幾條都是從服務邊界起算的，這一條補上
  「玩家自己走得到店裡」。
- 賣出（`cmd/pool-game/shop_sell_test.go`，全部從 `Update()` 送鍵）：
  `TestSellingThroughUpdatePaysHalfTheValue`（長劍 15 → 出價 7 → 白金 +1 金 +2，
  隊伍與角色庫同步）、`TestSellingDeclinedKeepsTheItem`、
  `TestSellingRefusesAReadiedItem`、`TestSellingIsNotOfferedForAMoraleNPC`、
  `TestSellingOverloadedPutsThePlatinumInThePool`、`TestSaleSurvivesSaveAndLoad`、
  `TestShopSaleMatchesTheDosgolemReceipt`。規則本身在
  `internal/treasure/sell_test.go`（除以 2、一疊除以 20、16 位元繞回、死分支、
  舊負重的超重判定）。

- 公款與鑑定（`cmd/pool-game/shop_pool_identify_test.go`，全部從 `Update()` 送鍵）：
  `TestLeavingTheShopWithPooledMoneyAsksFirst`（Y 回店、N 離店且公款不動）、
  `TestLeavingTheShopWithAnEmptyPoolDoesNotAsk`、`TestSharingThePoolInTheShopThenLeaving`、
  `TestIdentifyingThroughUpdateRevealsTheBonus`（`ITEM8.DAX/20h` 的長劍 → `Long Sword +1`，
  再鑑定一次照收 200）、`TestIdentifyingWithoutMoneyIsRefused`、
  `TestIdentifiedItemSurvivesSaveAndLoad`、`TestShopIdentifyMatchesTheDosgolemReceipt`；
  進店清公款由 `TestNormalKeysBuyAndEquipFromTheWeaponShop` 走進武具店之後斷言。
  規則在 `internal/treasure/identify_test.go`（311 筆內嵌名稱的正對照、`s ` 的分支、
  付款順序與拒絕）。

## 不做

- 不替 `6E6Ch`／`6EF6h` 取名。四家店都寫 1，目前只當邊界條件的一部分。
- 商店畫面的版面是 remake 的呈現；原版商店選單的**按鍵與分派**已讀（〈賣出〉），
  版面不宣稱一致。
- 名稱的 `* `（偵測魔法）與 ` Yes  `／` No   `（穿戴欄）前綴，見〈鑑定〉。
- 商店裡的 T）ake（挑幣別與數量）還沒接，公款只能 S）hare 回來。
