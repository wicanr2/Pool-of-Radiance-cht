# Goal：休息要挑地方——駕駛走去旅店付一枚白金（GitHub #38；spec 114／102／081／090）

主台帳：[issue #38](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/38)。
上一個 goal 的結案位置：[`issue-24-20-rest-entry2-and-clock.md`](issue-24-20-rest-entry2-and-clock.md)
（#24／#20 關：入口 2 接上、時鐘投影七位）。

#24 把入口 2 接上之後，「哪裡睡得著」第一次變成真的規則，而測試駕駛還不會挑地方。
**調查那一半已經做完了（下一節），這一輪只剩駕駛與治具。**

## 現況（2026-09-16，`edb168b` 已推送）

### 規則（exact，spec 114 已寫進去）

| 區 | 入口 2 | 週期／門檻 |
|---|---|---|
| 城區 `ecl3/0 9A5Eh → 9A63h` | `4ABA < 254` 且 `4A07 == 0` | 1／101（每一刻擲、必中）|
| 貧民窟 `ecl2/20 9A0Eh` | 街上（`@6E82 == 0`）、還沒清完 | 24／24 |
| 同上 | 屋內（`81h`／`82h`／`83h`，遮罩是 `7Fh` 不是 `1Fh`）| 0／0 |
| 古托井 `ecl8/29 9BDDh` | 地面 | 12／12 |
| 索寇要塞 `ecl4/21 9A29h` | 巡邏還在 | 2／1 |

回一點生命力要**連續**睡滿 24 小時（288 刻）。街上每兩小時擲一次、24% 中，十二次全過只有 4%
——**街上實際上睡不了**。而且貧民窟被打斷跑的入口 3（`9A49h`）不是選單，是
`SAVE 200 @4A1F`、`PARTYSTRENGTH`、`GOTO 9B68h` 直接排一場隨機遭遇；實測傷兵連睡八次就是全滅。

### `4A07` 是旅店的房錢（exact，2026-09-16 讀出來）

城區地形索引 9（spec 102）那七格——GEO3/0 的 `89h`：**(4,12) (6,12) (4,13) (6,13) (0,14) (1,14) (2,14)**，
從城門 (0,4) 走街道 **12 步**就到——跑 `ecl3/0 A140h`：

```
A140  COMPARE @4A07, 0 ; IF <> ; EXIT        ; 已經有房間就不再問
A14E  PICTURE 24
A15C  PRINTCLEAR "'IT WILL COST YOU 1 PLATINUM PIECE TO REST HERE.  DO YOU WANT TO STAY?'"
A195  GOSUB AE5A → A199 ON GOTO [A1A5, AA38] ; YES／NO（YES 是索引 0）
A1A5  GOSUB A1B4      ; WHO 'WHO WILL PAY?' → 扣一枚白金（`6BC3`，spec 090）
                      ;   不夠：A20Dh "YOU DON'T HAVE ENOUGH PLATINUM." 然後 EXIT
A206    SAVE 1 → 4A07 ; （在 A1B4 裡，收完錢才寫）
A1A9  GOSUB 9A63      ; ★ 城區入口 2 的本體，重算 6DD2／6DD3 → 這時 4A07 == 1，寫 0／0
A1AD  PROGRAM 9       ; 紮營（spec 081）
```

三件跟著出來的事：

1. **付完錢腳本自己重跑入口 2**（`A1A9h`），所以 `PROGRAM 9` 開的那次紮營是 0／0，睡得安穩。
2. **remake 早就接好了後半**：`ProgramCamp = 9` → `openCamp`（`program.go`），而 `openCamp`
   （#24 之後）又會跑一次入口 2，`4A07 == 1` 時答案一樣是 0／0——兩邊不打架，**不必改產品碼**。
3. **房間只管這一次**：城區地形索引 0（一般街道，185 格）跑的 `AE6Ah` 開頭就是 `SAVE 0 @4A07`，
   所以踏回街上房間就沒了，下次要再付一枚。

所以**這一輪不需要再讀原版**，缺的只有「駕駛走過去、答 YES、挑一個身上有白金的人付錢」。

### 駕駛與治具現況

- `restUntilHealed`（`cmd/pool-game/mainline_rest_test.go`）開頭：跑入口 2，`Period != 0` 就
  `note("...not sleeping here")` 直接 return。鏡像的 verify 就是這一句。
- 試過又退回的：「走去屋內找床」——傷兵在路上四場架就死了（#24 那一輪實測，與該函式先前註解
  記的教訓一樣）。旅店在**城區**，比貧民窟屋內遠，但路上是街道、沒有固定事件。
- `TestQuickCombatEarnsExperience` 是紅的：`newGameAtFirstCombat` 的治具在城裡漫無目的地走 900 步
  才找到第一場架（量到第 15 個小時，走到碼頭搭船去索寇打四隻毒蛙），時鐘接上之後城裡十四點宵禁的
  衛兵 38 隻（含 12 名六級戰士）先把它攔下來。治具的隊伍**沒有任何裝備**（AC 10、空手），所以
  改走貧民窟或索寇打隨機遭遇也打不贏（遭遇隻數跟隊伍強度放大，實測 12～16 隻）。

## 提示詞（可直接貼給 `/goal`）

> 目標：測試駕駛需要休息而這一區會打擾時，走去城區旅店付一枚白金睡滿；順手修好
> `TestQuickCombatEarnsExperience` 的治具。主台帳 #38。**上一節的 ECL 已經讀完，不要重讀**；
> 每一輪開跑前寫「這輪跟上輪差在哪」。
>
> 1. **駕駛（`mainline_rest_test.go`）。** 把「這一區會打擾就不睡」換成 `restAtTheInn()`：
>    (a) 不在城區就先走回城區（貧民窟往東 (15,4) 出界；古托井等區照各自的路，走不回去就維持
>    現在的「不睡、記一行」）；(b) 走街道到旅店七格之一（(4,12) 最近，從城門 12 步）；
>    (c) 踏上去、`YES`、`WHO WILL PAY?` 挑一個 `Money[Platinum] >= 1` 的人；(d) `PROGRAM 9`
>    會自己開紮營，接著照現在的排時間流程睡。全部走 `Update()` 送鍵。
> 2. **測試。** (a) 一條從 `Update()` 走的收據：帶白金的隊伍在城區受傷 → 走到旅店 → 付錢
>    （`4A07` 由 0 變 1、那個人的白金少一枚）→ `6DD2`／`6DD3` 變 0／0 → 睡滿、HP 回滿；
>    (b) 負對照：全隊白金不足時印 "YOU DON'T HAVE ENOUGH PLATINUM." 且 `4A07` still 0；
>    (c) 負對照：睡完踏回街上一步，`4A07` 被 `AE6Ah` 清回 0。
> 3. **治具（`playthrough_test.go`）。** `newGameAtFirstCombat` 要的是「一場打得贏的架」。
>    先給隊伍裝備（照 `outfitParty` 那條路買，或直接在治具裡塞武器與甲），再讓它走一條有目的
>    的路找架；**不要把時鐘關掉**，也不要為了閃宵禁而改規則。修好之後 `TestQuickCombatEarnsExperience`
>    與 `TestPassiveCombatTerminates` 都要綠。
> 4. **不可越線：** 不改腳本、不改 engine／CoAB、不給錢（白金要是隊伍自己賺的）；產品碼這一輪
>    預期**不用動**（`PROGRAM 9` 已接），真的要動先說明為什麼。
> 5. **台帳：** 每輪在 #38 留言；spec 114 的狀態行；駕駛學會之後在 #22 留言（策略層多一手）；
>    收尾 `docs/worklist.json`（#38 的 verify 是 `mainline_rest_test.go` 裡的 `not sleeping here`，
>    拿掉就會翻）→ `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open`。
> 6. **停止線：** 走回城區的路在某一區走不通（記下那一區，維持「不睡」）；治具給了裝備還是打不贏
>    （記下打的是哪一場、命中率多少，另開 issue）；旅店那一格踏上去沒有反應（那是產品缺陷，先修）。

## 已知風險與待決

- 旅店在城區，貧民窟打完要走回 (15,4) 出界再穿過城區——路上會再撞遭遇。傷兵走不回去是常態，
  所以 (a) 那一步要保留「走不回去就不睡」的退路，不要硬走（#24 那一輪就是這樣死的）。
- 一枚白金一晚，而房間踏回街上就沒了：連睡兩晚要付兩枚。探針的隊伍開場有白金，但長期要靠委任
  獎金（#37 的槽 0 是 250 金＋200 白金）。
- 治具給裝備會改動骰流，`TestPassiveCombatTerminates` 打的那一場也會換——那條只要求戰鬥會結束，
  預期還是綠，但要一起跑。
