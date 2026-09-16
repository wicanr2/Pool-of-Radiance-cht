# Goal：休息要跑各區的 ECL 入口 2、時鐘要投影到 `49C6h..49CCh`（GitHub #24、#20；spec 114／069／102）

主台帳：[issue #24](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/24)（休息不會被打斷）、
[issue #20](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/20)（時刻沒投影，白天判斷永遠成立）。
上一個 goal 的結案位置：[`issue-37-kuto-well-norris.md`](issue-37-kuto-well-norris.md)
（#37 關：古托井四題判完、探針走到獎金買板甲與 L2）。

兩條放同一輪：都是「引擎這一側推了時間，卻沒把結果交給 ECL」。#24 缺的是每一區的入口 2，
#20 缺的是七個時鐘變數；而且它們互相咬——休息推時鐘，時鐘決定城門的馬車商人出不出現
（spec 137 死區表），而要撐過 14:00 得靠休息。

## 現況（2026-09-16，`c83e108` 已推送）

### #24：入口 2 就是每一區的打斷參數，remake 一次都沒跑過

`camp.go` 的 `restInterruption()` 從 `eventMachine.Memory` 讀 `6DD2h`（週期）／`6DD3h`（門檻），
spec 114 寫著「ECL VM 本來就在跑那些 `SAVE`，所以值是自然到位的」——**那句話是錯的**。
寫這兩個位址的 `SAVE` 幾乎都在**入口 2**，而 remake 只跑入口 0（每格）、1（搜尋）、3（打斷之後）。
`grep SetEntry cmd/pool-game/camp.go` 一筆都沒有，所以兩個值永遠是 0／0，任何地方休息都不會被打擾。
（spec 114 那一句要照 `rulebook/63` 改成現況，推翻紀錄集中一處。）

入口 2 的位址從 `pool-ecl-trace` 的 `entry_addresses` 取（索引 2），已經讀出來的四份：

| 區 | 入口 2 | 內容 |
|---|---|---|
| 城區 `ecl3/0` | `9A5Eh` → `GOSUB 9A63h` | `4ABA < 254`（還沒通關）且 `4A07 == 0` → `6DD2 = 1`、`6DD3 = 101`；否則 0／0 |
| 貧民窟 `ecl2/20` | `9A0Eh` | `4A0B == 255` → **24／24**；否則 `4ABB >= 254`（清完）→ 0／0；否則地形 `@6E82 == 0`（街上）→ **24／24**；否則 0／0 |
| 古托井 `ecl8/29` | `9BDDh` | 先 `CALL @C018`、預設 `0／100`；井底（`4A10 == 1`）地形 16 就 EXIT，諾里斯還活著（`4A24 != 255`）→ `6DD2 = 1`；地面 `4A22 < 10` 且地形不是 1 → `12／12` |
| 索寇要塞 `ecl4/21` | `9A29h` | `4A03 <= 4`（巡邏還在）→ `2／1`；否則 0／0 |

**入口 2 讀地形**（`@C04F`、`@6E82`），所以跑之前要先投影座標——`gamepack.RunInitialSessionCampEntry`
（入口 3）已經有 `projectInitialPosition` 那一步，入口 2 照抄。

**還沒讀到的**：原版在哪一刻叫入口 2。spec 114 已證休息主迴圈 `0C45h` 裡**沒有**任何 ECL 呼叫
（`0D69`／`0D6D`／`0D76` 是 entry 11／14／15），所以它一定在迴圈之前——候選是進紮營畫面、
按下 R）EST、或每次換格／換圖。這是這一輪要判的第一件事。

### #20：七個時鐘變數一個都沒投影

spec 069〈時間推進〉已證：**overlay-20 entry 2（`0392h`）把時間加到 ECL 的
`49C6h..49CCh`**（class 0，`[4933h] + 6E00h + addr × 2`），entry 4 再換算成分、遞減效果。
remake 的 `advanceGameTime`（`party_panel.go:155`）只動 `a.gameTime` 與效果，沒有那一步。

索引對位址（spec 114 的進位表）：`49C6h` 起 0..6，所以 **`49C9h` 是小時**（上限 24）、
`49C8h` 是分的十位、`49CAh` 是日。腳本讀的就是 `49C9h`：

| 讀的地方 | 條件 | 效果 |
|---|---|---|
| `ecl3/0 9BAEh` | `49C9 > 14` 就跳過 `PICTURE 41` | 上船的圖 |
| `ecl3/0 ADAAh` | 城門的馬車商人 | `49C9 >= 14` 直接 EXIT（spec 137 死區表）|
| `ecl4/21 AE48h` | `49C9 >= 14` → `4A18 = 8`（否則 11）| 索寇要塞的圖 |
| `ecl8/29 AF30h` | 同類 | 古托井 |

`49C6h..49CCh` 的引用清冊用 `cmd/pool-ecl-memory-audit -addresses 49C6,49C7,49C8,49C9,49CA,49CB,49CC`
重生，跑之前先帶一個已知答案當正對照。

## 提示詞（可直接貼給 `/goal`）

> 目標：休息時跑目前這一區的 ECL 入口 2，讓 `6DD2h`／`6DD3h` 有真值；時鐘每次推進都投影到
> `49C6h..49CCh`。主台帳 #24、#20。上一節的入口 2 位址與四份內容已經讀出來了，不要重讀。
>
> 1. **先讀原版，不改 remake。** 只有一個問題：**入口 2 是誰在什麼時候叫的**。spec 114 已證休息
>    主迴圈裡沒有 ECL 呼叫，所以往上一層找：紮營畫面的進入點、`R)EST` 的處理常式、或換格／換圖
>    的 lifecycle（入口 0／1 的呼叫端在哪，入口 2 多半就在旁邊）。用 `tools/ida.sh` 讀 overlay-20
>    的紮營那一段與它的呼叫端；正對照是入口 3 的呼叫點（spec 114 已經定位）。判不出來就用
>    dosgolem：在貧民窟街上與屋內各紮營一次，`-peek` 讀 `6DD2h`／`6DD3h` 換算後的引擎位移
>    （`[4937h] + 5A4h`／`5A6h`），看值在哪一刻從 0 變成 24——那一刻的前一個動作就是呼叫點。
>    結論寫進 spec 114（把「值是自然到位的」那一句改掉，推翻紀錄集中一處，見 `rulebook/63`）。
> 2. **改 remake（#24）。** 照第 1 點的時機加一支 `RunInitialSessionRestEntry`（entry index 2，
>    先 `projectInitialPosition` 再 `SetEntry(2)`，與 `RunInitialSessionCampEntry` 同形）；
>    `camp.go` 在算 `restInterruption()` **之前**叫它。入口 2 有可能吐事件（`CALL @C018` 那類），
>    照入口 3 的消費流程接，不要另寫一份。
> 3. **改 remake（#20）。** `advanceGameTime` 之後把 `a.gameTime` 的七位寫進
>    `eventMachine.Memory[0x49C6..0x49CC]`；讀檔（`poolsave` 還原）與換圖（`NEWECL`／`configureEventSession`）
>    之後也要投影一次，否則新的 machine 是 0。**不要只寫 `49C9h`**——原版寫七個，少寫的那幾個
>    會在別的腳本上變成安靜的錯誤。
> 4. **測試都從 `Update()` 送鍵。** (a) 貧民窟街上紮營會被城衛隊打斷、同一區屋內不會（負對照）；
>    (b) 城區紮營的 1／101 與清完貧民窟之後的 0／0 各一條；(c) 時鐘推到 14 點之後城門的馬車商人
>    不出現（#20 的驗收條件），推回早上又出現；(d) 存讀檔一次，確認投影還在。
> 5. **不可越線：** 不改腳本、不改 engine／CoAB、不碰 `6DD2h`／`6DD3h` 的值（那是腳本自己寫的，
>    spec 114 已證入口 3 開頭就把門檻清掉）；動到畫面才跑對拍。
> 6. **每輪開跑前寫「這輪跟上輪差在哪」**；先一區（貧民窟街上）再推到其他區。
> 7. **台帳：** 每輪在 #24／#20 留言；spec 114／069／102 狀態行；探針若因此多過一段就在 #22／#5
>    留言；收尾 `docs/worklist.json`（兩條的 verify 都是 `absent`：`camp.go` 的 `SetEntry(2)`、
>    三個檔的 `0x49C9`，接上就會翻成「做好了」）→
>    `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open`。
> 8. **停止線：** 入口 2 的呼叫點在 overlay 與 dosgolem 兩邊都判不出來（另開 issue 擴掃描面）；
>    投影之後某個腳本開始走不同的分支而那個分支沒有原版證據（記下、標 hypothesis、不要硬接）。

## 已知風險與待決

- 入口 2 若其實是「每次換圖就跑一次」而不是「休息前跑」，那 #24 的接點在 lifecycle 不在 `camp.go`，
  worklist 那一條的 verify（`camp.go` 的 `SetEntry(2)`）要跟著改成新的位置。
- 投影時鐘會讓 `ecl3/0 9BAEh`、`ecl4/21 AE48h` 這些 `PICTURE` 分支開始變動，畫面可能因此不同——
  那是修好之後的正確行為，但**要跑對拍並把數字寫進 commit**（CLAUDE.md §7）。
- 探針目前在城區的時刻一直是 0（等於永遠早上），接上之後它的路線可能要多一次紮營；主線探針
  那兩條本來就紅，別把它們的變化誤判成這一輪的迴歸——先看的是新測試。
