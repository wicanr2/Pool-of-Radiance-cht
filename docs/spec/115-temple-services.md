# Spec 115：神殿的九項服務（overlay-04）

狀態：READY（九項的名稱、價錢、前提、付款方式與各自拿掉哪些效果都已讀出、
實作並接上界面；起死回生的體質與生命力重算也接了）。
日期：2026-09-05（2026-09-10 拿掉 `+11Bh` 那個 OPEN——它是目前生命值，
本規格第 30 與 45 行自己就這樣註解，spec 097 的能量吸取段與
`MonsterRecord.CurrentHitPoints()` 也讀同一格）。

## 選單

神殿的選單列有兩個版本，差別在地上有沒有錢可以撿：
`Heal View Take Pool Share Appraise Exit`（`0C1Ah`）與
`Heal View Pool Appraise Exit`（`0C42h`）。H）EAL 底下就是這九項。

說明書 p.34 對神殿只寫兩項：「進入神殿後，各種處理錢財的指令都與在商店中的
一樣，不再贅述，因此只介紹二項選擇項」——H）EAL 與 E）XIT。所以錢財那幾項
（View／Take／Pool／Share／Appraise）與商店共用。

## 九項服務

每一項的形狀相同：先問「這個人有沒有那個毛病」，沒有就印 `is not …` 並讓玩家
確認要不要照做；接著印名稱、報價、由 **entry 3（`00BFh`）**收錢；付了才處理。

| 進入點 | 服務 | 價 | 前提 | 付款後 |
|---|---|---:|---|---|
| entry 4 `0249h` | Cure Blindness | 1000 | 效果 `21h` | 拿掉 `21h` |
| entry 5 `02EDh` | Cure Disease | 1000 | `DS:0112h..0117h` 六個代碼任一 | 拿掉命中的 |
| entry 6 `0401h` | Cure Light／Serious／Critical Wounds | 100／350／600 | 無 | 擲骰治療（spec 既有）|
| entry 7 `0515h` | Raise Dead | 5500 | `+10Ch` 是 6 或 1 | 見下 |
| entry 8 `0736h` | Neutralize Poison | 1000 | 效果 `37h` | 拿掉 `37h`／`16h`／`0Fh` |
| entry 9 `081Fh` | Remove Curse | 3500 | 身上有 `+36h` 非 0 的物品（`0845h` 先看）或效果 `24h`（`0888h`）| 交給 overlay-22 entry 9（`2508h`：有 `24h` 只解它，否則清第一件被詛咒的物品，spec 098，#66）|
| entry 10 `0907h` | Stone to Flesh | 2000 | `+10Ch` 是 7 | `+10Ch = 0`、`+10Dh = 1`、`+11Bh = 1`（生命力回到 1）|

疾病那六個代碼是 `1Fh 22h 2Bh 2Ch 32h 39h`。原版讀的是 `[di+111h]`、索引
1..6——那是 Turbo Pascal `array[1..6]` 的**偏移基底**寫法，元素其實從 `0112h`
起（同 spec 112 的 `DS:6786h`）。`DS:0111h` 那個 byte 不屬於這張表。

`+10Ch` 的狀態值與 `combat.DeadState` 對得上：6 是死亡，7 是石化，
8 是被轉變不死生物摧毀（spec 111）。

## 沒有毛病照樣收錢

（2026-09-27，#118，exact。overlay-04 code SHA-256 `d948ce6bc533470ac1fa44a7787c2ce5006462cc33ada1cf61ffd128da8a91af`，
`coab-go-test:20260729` 的 GNU objdump，位址是 overlay 檔內位移。）

有前提的六項（失明、疾病、起死回生、解毒、除咒、石化解除）形狀都一樣：沒有那個毛病就印
`is not …`，交給 `0013h` 問一句 "cast cure anyway: "（`0000h` 的 Pascal 字串
`12 63 61 73 74 20 63 75 72 65 20 61 6E 79 77 61 79 3A 20`），`011Dh:003Eh` 讀一個鍵回傳；
是 `'Y'` 就照樣報價、由 entry 3（`00BFh`）收錢。除咒（entry 9）是這樣：

```
082B  C6 86 F6 FE 59            答案預設 'Y'（有毛病就不問）
0891  8D BE E7 FE 16 57 BF 03 08 0E 57 9A 34 06 BB 05   ; "is not cursed."（cs:0803h）
08A1  0E E8 6E F7               call 0013h("cast cure anyway: ")
08A5  88 86 F6 FE               答案 = 讀到的鍵
08A9  80 BE F6 FE 59 75 35      不是 'Y' → 收工
08C0  B8 AC 0D 50 0E E8 F7 F7   00BFh(3500)：報價、問 pay for cure、收錢
08CC  80 BE F6 FE 59 75 12      沒付 → 收工
08D3  DS:6B89h = 這個人；08E0 9A 4D 00 E2 00   overlay-22 entry 9（`2508h`）
```

付完的處理對沒有毛病的人什麼也不做：失明、疾病、解毒是拿掉不存在的效果碼；除咒的 `2508h` 兩樣都
找不到；起死回生在 `05A5h`（`80 BE FA FE 00 75 03 E9 60 01`）看付款前記下的「真的死了」旗標，
石化解除在 `096Ah`（`26 80 BD 0C 01 07 75 1E`）重讀 `+10Ch`，不符都直接收工——**活人付了 5500
也不會被改成 1 點生命、扣體質**。三種傷藥沒有前提，本來就照收。

remake：`temple.Serve` 先記下 `Applies`，照付；沒有毛病就付完直接回傳。選單上兩問（anyway 與
pay for cure）併成一個 YES／NO，`is not …` 那一句放在狀態列（停止線，呈現）。測試
`TestServiceChargesEvenWhenThereIsNothingToCure`、`TestRaiseDeadOnTheLivingOnlyTakesTheMoney`、
`TestTempleChargesEvenWhenThereIsNothingToCure`（從 `Update()` 按鍵）、
`TestTempleRemoveCurseChargesTheUncursed`（同上）。

## 起死回生要付一點體質

```
05af  DS:677Dh = 1                       ; 抑制旗標
05c5  移除效果 20h；05db 移除效果 37h
05f3  +10Ch = 0；05fd +10Dh = 1
05e9  +11Bh = 1                          ; 目前生命值——活過來只剩一點
0607  +14h（體質）減一
060f  差額 = +32h（生命力上限） − +B1h（不含體質加成的份）
0631  體質 < 0Eh（14）→ 收工
063b  對職業槽 i = 0..7，等級 > 0 時：
0659    i == 2（戰士）→ 權重 += 等級 × (體質 − 14)
0688    否則 體質 > 0Fh（15）→ 權重 += 等級 × 2
06ac    否則              → 權重 += 等級
06ca  每級 = 差額 ÷ 權重                  ; 整數除
06e3  體質 >= 11h（17）而且有戰士等級 → 收工（不扣）
06f6  +32h −= 每級；+11Bh −= 每級
```

體質是**扣掉一點之後**的值：`0607h` 先減，`0677h` 才讀。所以復活的代價是
「體質 −1，生命力上限跟著少掉一個生命骰的體質加成」，而且**復活時只有 1 點
生命力**。

`+11Bh` 是目前生命值：overlay-09 的士氣判定拿 `+11Bh × 100 ÷ +32h` 當
「還剩幾成血」（spec 096），那是它的用途證據。

## 付款

**entry 3（`00BFh`）**：先看這個人自己的金幣等值（五種硬幣，只取低位字），不夠才由整隊公款出全額，
**兩邊不合併**；付完兩邊都重鑄成白金＋金（`treasure.PayGold`，spec 067〈公款〉）。字串是 ` will only cost `／` gold pieces.`／`pay for cure `／
`Not enough money.`／`is cured.`。

## 實作

`internal/temple`：`Services` 是九項的資料表，`Serve()` 收錢並套用；三種傷藥
仍走既有的 `CureWounds`（那三支要擲骰）。效果碼存在存檔的 `Character.Effects`
（spec 069 的串列），狀態存在 `Character.Status`（`+10Ch`）。
起死回生的重算是 `raiseDeadHitPointLoss`。

選單那九項由 `templeHealServiceIDs` 排序、名稱從 `temple.Services` 取，
所以兩份表不會漂開（`TestTempleMenuCoversEveryService` 釘住）。
A）ppraise 見 [spec 116](116-appraise-and-sell.md)。

## OPEN

- `DS:677Dh` 那個在移除效果前後立起又放下的旗標。
