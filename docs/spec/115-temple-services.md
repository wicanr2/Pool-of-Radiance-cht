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
| entry 9 `081Fh` | Remove Curse | 3500 | 效果 `24h` | 交給 overlay-22 entry 9 |
| entry 10 `0907h` | Stone to Flesh | 2000 | `+10Ch` 是 7 | `+10Ch = 0`、`+10Dh = 1`、`+11Bh = 1`（生命力回到 1）|

疾病那六個代碼是 `1Fh 22h 2Bh 2Ch 32h 39h`。原版讀的是 `[di+111h]`、索引
1..6——那是 Turbo Pascal `array[1..6]` 的**偏移基底**寫法，元素其實從 `0112h`
起（同 spec 112 的 `DS:6786h`）。`DS:0111h` 那個 byte 不屬於這張表。

`+10Ch` 的狀態值與 `combat.DeadState` 對得上：6 是死亡，7 是石化，
8 是被轉變不死生物摧毀（spec 111）。

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

**entry 3（`00BFh`）**：先看這個人自己的金幣，不夠才由整隊公款出全額，
**兩邊不合併**。字串是 ` will only cost `／` gold pieces.`／`pay for cure `／
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
