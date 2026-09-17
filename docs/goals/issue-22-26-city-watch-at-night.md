# Goal：城區 (3,4) 全滅的觸發鏈對回原版，探針走到索寇要塞（GitHub #22、#26 → #5）

主台帳：[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（策略層）、
[issue #26](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/26)（路線與建議順序），
兩條做完後回 [issue #5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)。
上一個 goal 的結案位置：[`issue-38-rest-at-the-inn.md`](issue-38-rest-at-the-inn.md)
（#38 關：駕駛會走去旅店付一枚白金睡滿、睡到早上）。

## 現況（2026-09-17，`ff2098e` 已推送，工作樹乾淨）

- open issues 8 條（#5 #6 #17 #19 #21 #22 #23 #26），`docs/worklist.json` 8 條對得上，
  `pool-worklist -mode verify` 全部回「仍未完成」。
- house rule 探針（`TestMainlineProbeHouseRuleCommissionExperience`，seed 143）打贏諾里斯
  （`4A24=FF`、槽 0 `FE`）、用到旅店，**走回市政廳時在城區 (3,4) 全滅**
  （`docs/playtest/mainline-end-to-end.md` 補八）。交接把成因記成「十四點宵禁、38 隻衛兵」。
- 全套測試上一輪記為五條紅（索寇、世界巡迴、自己玩、兩條主線探針）；**這一份提示詞
  寫成時沒有重跑**，開工第 0 步自己量。

### 「十四點宵禁」這個前提沒有證據，靜態 trace 指向別的機制

「城裡十四點宵禁、衛兵 38 隻」出現在 `cmd/pool-game/mainline_rest_test.go`（`restMorningHour`
註解、睡到早上那一段）、`cmd/pool-game/playthrough_test.go`（`equipForFirstCombat` 與治具
走出城區那一段）、spec 114〈睡覺要睡到早上〉、playtest 補八，以及 #24／#38 的 goal。引用的
證據是 `ecl3/0 9BAEh`（碼頭的 `PICTURE 41`）與 `ecl3/0 ADAAh`（註解說是馬車商人）——**兩處都不是戰鬥**，
而且 `ADAAh` 在 `ecl3/0` 裡落在 `AD86h` 那句 PRINTCLEAR 的字串中間（offset 5290，介於指令 5254 與 5315），
不是一條指令；馬車商人在 `ECL2/9`（spec 137 第 8 段）。

`workplace/ecl3-block0.json`（`ecl3.dax` SHA-256 `db58f0c6…`，block 0 `b0fe79c5…`，
code base `0x9900`）讀出來的是：

```
入口 0（每一步）
9914  SAVE 0 → 6E7B ; SAVE 11 → 6E7D
9920  COMPARE @49C9, 14 ; IF >= ; SAVE 8 → 6E7D     ; 方向待對（spec 102 9BAEh 的讀法）
992D  SAVE @6E7D → 49FD ; SAVE 10 → 49FE             ; 49FD／49FE 是什麼：未讀
9965  AND 127, @C04F → 6E82                           ; C04F = 格子的原始 terrain byte
9976  COMPARE @6E7D, 8 ; IF <> ; EXIT                ; 只有「晚上」才往下
997E  (6E82, C04E) ∈ {(0,9), (4,7), (26,11)} → GOTO 99AFh   ; C04E 的意義與三格對到哪：未讀
99AF  "THE DOOR IS LOCKED.  DO YOU WANT TO BREAK IN?"  YES／NO（AE5Ah）
99D8  ON GOTO [AD82h ← YES, 99E4h ← NO]

AD82  "THE CITY WATCH RESPONDS TO THE NOISE.  DO YOU STAY TO FIGHT THEM OR RUN AWAY?"
ADC3  HORIZONTAL MENU [STAY, RUN] → ADD3 ON GOTO [ADDFh, AEF4h]
ADDF..AE08  LOAD MONSTER 94×2、84×12、53×12、40×12 → COMBAT    ; 38 隻就是這一場
```

`AD82h`／`ADDFh` 的進入邊一共五條，**每一條都是玩家選了才開打**，沒有一條是時間到自動出兵：

| 來源 | 問句 | 開打的選項（索引） | 安全的選項 |
|---|---|---|---|
| `99D8h` | 夜裡的鎖門 BREAK IN? | YES（0）| NO |
| `9AE6h` | 休息被驅趕 ROUSTED BY THE CITY WATCH | STAY（1）→ `ADDFh` | GO → `AE6Ah` |
| `9E9Ah` | 神殿衛兵擋主教 | 其中一項 → `AD82h` | 另一項 → `A059h`（未讀）|
| `A828h` | 酒館鬥毆後 CITY WATCH CHARGES IN | 其中一項 → `AD82h` | `AEF4h` |
| `AFDCh` | 瘋子發作 | 直接 `GOTO AD82h` | — |

證據等級：分支結構是 **exact**（靜態 trace、位址與運算元逐條可查）；「`49C9 ≥ 14` 是晚上、
那三格是晚上會鎖的門」是 **strong inference**；「(3,4) 全滅就是鎖門答 YES → STAY」是
**hypothesis**——但選單第 0 項剛好是 YES／STAY，駕駛只要照預設按 ENTER 就會一路開打，
這是目前最可能的一條。

## 提示詞（可直接貼給 `/goal`）

> 目標：查清 house rule 探針（seed 143）在城區 (3,4) 全滅是哪一支 ECL 分支觸發的，把
> 「十四點宵禁」這個前提改成原版機制，駕駛照原版分支避開；然後讓探針走過交件 → 訓練所升級
> → 買甲 → 索寇要塞 (8,5)，記下每一段的結果。主台帳 #22、#26，完成後回 #5。
>
> 0. **開工基準。** `git log -5`、`gh issue list --state open`、`pool-worklist -mode verify`；
>    跑一次全套測試記下紅燈名單與耗時（上一輪說五條紅，不要沿用，自己量）。
> 1. **先重現，不改駕駛。** 跑 `TestMainlineProbeHouseRuleCommissionExperience`，在全滅前
>    印出最後三個格子事件的 ECL 位址、畫面文字、選單選項、送出的鍵、座標與 `49C9`。
>    對照〈現況〉那張表確認是哪一條進入邊；不是表上任何一條就停下來回報。接著寫一條
>    0.x 秒的最小重現：從正常開局把時鐘排到晚上（紮營的小時欄，不注入 `49C9`）、走到那一格、
>    斷言問句出現、答 NO 不開打。
> 2. **把機制讀完，寫進規格。** 讀 `ecl3/0` 入口 0 `9914h..99EAh`：`COMPARE @49C9, 14`
>    的方向（拿 spec 102 `9BAEh` 的讀法對一次）、`C04E` 是什麼欄位、三組 `(6E82, C04E)`
>    對到 GEO3/0 哪幾格（市政廳在不在裡面）、`49FD`／`49FE` 被誰讀；再讀 `9E9Ah` 與
>    `A828h` 的選項文字與另一個目標。結論寫進 spec 102（城區派發）或 spec 114，每條標證據
>    等級。runtime 證據能用 dosgolem 在原版夜裡站上那一格就做一次；做不到就標 strong inference。
>    **同時核對 remake 有沒有照這條跑**（夜裡鎖門、答 YES 才出兵）；行為不同就先開 issue
>    帶證據，不在這一輪順手改產品碼。
> 3. **改掉錯的前提文字。** 上列程式註解、spec 114 與 playtest 的「十四點宵禁、到點出兵」改成原版機制
>    （#24／#38 的 goal 是歷史文件，不改），`ecl3/0 ADAAh` 這個引用一併查清楚它原本指的是哪一區哪一條；推翻紀錄只寫在
>    `CONTEXT.md` 的被推翻斷言表一處，正文只寫現況。`restMorningHour`／`hoursUntilMorning`
>    留不留依第 2 步決定——夜裡鎖門仍會擋住進店、進市政廳，理由要改寫成那一條。
>    `TestMainlineProbeHouseRuleCommissionExperience` 的註解還寫 seed 137，實際是 143，一併改對。
> 4. **駕駛照選項文字答，不靠預設 ENTER。** 只動測試治具：鎖門問句答 NO；被驅趕答 GO；
>    衛兵 STAY／RUN 答 RUN；要進的地方晚上會鎖，就先在旅店睡到早上再進。每條規則配一條
>    從 `Update()` 送鍵的測試。四條探索器測試（`TestRandomWalk…`、`TestSokalKeep…`、
>    `TestWorldTour…`、`TestPlayingTheWorld…`）也檢查是不是撞到同一條，是的話套同一個 helper。
> 5. **探針往下跑。** house rule 探針：諾里斯 → 市政廳交件 → 獎金集中給一個人 → 訓練所升級
>    （照職業挑門）→ 武具店用隊伍 pool 買板甲 → 索寇要塞 (8,5)。原版規則探針
>    （`TestMainlineProbeNaturalPartyFirstBattle`）也重跑一次。每段記等級、XP、金幣、AC、
>    時刻與結果，接進 `docs/playtest/mainline-end-to-end.md` 新的一節（寫「這輪跟上輪差在哪」）。
>    駕駛一改骰流就變：seed 要重掃並把勝率記下來，不挑一個會贏的 seed 當收據。
> 6. **路線表補時刻。** spec 137〈建議順序〉每一段補「大約耗幾小時、進城前要什麼時刻條件」，
>    依據是第 5 步的實跑數字；還沒實跑的段落維持 hypothesis。
> 7. **不可越線：** 不注入旗標、座標或時鐘；不 forced-win；不改能力值或金幣；不動
>    `golden-box-remake-engine` 與 CoAB；不為了讓測試過改判定規則。全滅是合法結果，記錄在哪一場、
>    什麼等級。建置測試走 `tools/go.sh`，`fmt` 不動既有檔；不准加大 timeout；治具迴圈一律帶 guard。
> 8. **台帳：** 每輪在 #22 留言（指令、走到哪、死因、下一個最小工作）；第 1～3 步做完在 #26
>    留言並更新 spec 137；探針打過索寇要塞 (8,5) 或同一死因第三次時，在 #22 寫結論，決定關或留。
>    順手更新 `docs/worklist.json` 裡 #5／#22／#26 的摘要——#26 那條還寫著「等級靠委任獎賞、
>    獎賞表還沒讀成表」，而 #26 留言早已讀出「委任只給錢、不給經驗值」。改完跑
>    `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open`，兩邊對齊。
> 9. **停止線：** 第 1 步的觸發點不在表上；同一場、同一死因第三次；要動 `internal/combat`
>    的判定才過得去；或一條探針超過 60 秒——停下來回報。

## 已知風險與待決

- `COMPARE` 的比較方向若讀反，「晚上」會變成「白天」，第 4 步的「睡到早上」就整個反過來。
  第 2 步一定要先拿 `9BAEh` 已經定案的讀法對一次，不要直接照上表的註解用。
- (3,4) 全滅若不是鎖門那一條（例如旅店回程紮營被驅趕答了 STAY），第 4 步仍然適用，但
  第 3 步要改寫的內容不同；以第 1 步印出來的位址為準。
- 索寇要塞 (8,5) 的 50 隻避不掉（#37 定論）。這一輪能不能打過取決於等級與戰術，
  不取決於這一份處理的城區問題；打不過就照停止線記錄，不在這一輪調戰術駕駛。
- 靜態 trace 用的是 `workplace/ecl3-block0.json`，這份檔案不在版控，引用前先用
  `cmd/pool-ecl-trace` 重生並核對 `block_sha256`。
