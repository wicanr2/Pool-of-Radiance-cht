# Goal：主線路線補「委任獎賞、建議等級與順序」，探針照順序跑（GitHub #26、#22）

主台帳：[issue #26](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/26)（路線與等級）、
[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（策略層剩餘項）。
上一個 goal 的結案位置：[`issue-22-player-strategy-layer.md`](issue-22-player-strategy-layer.md)。

## 現況（2026-09-15，commit `5d54161` 已推送）

策略層（治具）已有：包紮、催眠、集火、繞同伴、就地紮營重記、照說明書建隊、買甲、
開鎖門、不殺 NPC、避開打不起的固定事件。主線探針：貧民窟 20 場零死亡 → 存讀檔 →
索寇要塞，第二場 50 隻全滅。「20+ 隻的固定事件」全滅三次（33／24／50），觸發停止線。
經過在 [`docs/playtest/mainline-end-to-end.md`](../playtest/mainline-end-to-end.md) 補四／補五。

判定：缺的是**路線順序與等級**，不是駕駛。散戰 21 場只有 221 XP（二級要 1500～2000，
spec 071）；等級來自委任獎賞，而 `ecl3/8` 每槽的獎賞（`TREASURE 0 0 0 @6E7B..@6E7F`）
還沒讀成表。

## 提示詞（可直接貼給 `/goal`）

> 目標：spec 137 補一欄「建議等級／順序」，主線探針照那個順序以正常按鍵跑到能升級、
> 再回頭清貧民窟 25 場（`4ABB == FEh`）與索寇要塞；主台帳 #26、#22。
>
> 1. **先讀資料，不跑探針。** `ecl3/8` 每個委任槽的獎賞金／經驗讀成表（`cmd/pool-ecl-trace`
>    讀 `GETTABLE` 的表，先拿反組譯器已解出的指令對一次位址基準，CLAUDE.md §5）；每個委任區
>    列「最大一場幾隻、什麼怪、HD」（各區 `LOAD MONSTER` 清冊）。證據等級逐條標。
> 2. **算出順序。** 用 spec 071 的門檻與第 1 步的獎賞，排出「一級隊伍打得起 → 交件 →
>    訓練所升級 → 下一區」的順序，寫進 spec 137 新的一欄；每一步寫入口旗標與出口旗標。
> 3. **探針改照順序跑。** 訓練所（spec 097 `trainMember`）走正常按鍵；每段記等級、XP、
>    最大一場、結果。神殿治療付得起才用。有 XP 過門檻就先去訓練所。
> 4. **每輪開跑前寫「這輪跟上輪差在哪」；先跑最小重現**（單區、單場）。結果表接進
>    `docs/playtest/mainline-end-to-end.md`。不准加大 timeout。
> 5. **不可越線：** 不注入旗標／座標、不 forced-win、不改能力值或金幣、不動 engine／CoAB、
>    不為讓測試過而改判定規則（發現 remake 與原版不符 → 先開 issue 帶證據）。全滅是合法
>    結果，記錄在哪一場、什麼等級。建置測試走 `tools/go.sh`，`fmt` 不動既有檔。
> 6. **台帳：** 每輪在 #26 留言；路線欄寫進 spec 137 後關 #26；探針依順序打到貧民窟 25 場
>    ＋索寇要塞交件後在 #22 留言並回 #5 續跑。關閉後同步 `docs/worklist.json` →
>    `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open` 盤點。
> 7. **停止線：** 同一區、同一死因第三次；或發現要動 `internal/combat` 的判定；或一條探針
>    超過 60 秒——停下來回報。

## 已知風險與待決

- 獎賞表的 `GETTABLE` 位址基準偏一格會安靜地給出一張看起來合理的表（spec 136 踩過）；
  第 1 步要先對已知指令。
- 一級隊伍能做的委任可能不只一條路（古托井、墓園、圖書館書、波多廣場拍賣…）；
  順序由獎賞／風險比決定，寫清楚為什麼選這條。
- remake 的遭遇 staging 疑似有上限（36 隻的固定遭遇盤面上只有 10 隻，#22 留言）；
  若順序的可行性靠這個上限，要先把它對回 spec 048 再定案。
