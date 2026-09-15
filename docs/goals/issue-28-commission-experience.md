# Goal：自訂規則「委任折算經驗值」，再讓探針在它開著時往下走（GitHub #28、#22）

主台帳：[issue #28](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/28)（house rule）、
[issue #22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（策略層剩餘：訓練所、買甲、神殿）。
上一個 goal 的結案位置：[`issue-26-route-levels.md`](issue-26-route-levels.md)（#26 留著，路線表已補，
順序待實跑）。

## 現況（2026-09-15，commit `d6f4aab` 已推送）

- 原版委任只給錢（spec 137 獎賞表，exact）；經驗值只從怪物身上發（spec 097）；
  升級要 1000 金／級（說明書）。一級隊伍在原版就是靠催眠與位置去打 20～50 隻的大場。
- Q）UICK 已接上（spec 139，#27 已關）：全隊交給電腦打，經驗值照發。
- 使用者決定（2026-09-15）：加一條可切換、預設關的 house rule——委任獎賞折算經驗值，
  出處 AD&D 一版 DMG「寶物 1 gp ＝ 1 XP」（SSI 沒實作）。
- 探針現況：說明書建議的一級隊伍貧民窟 20 場零死亡 → 存讀檔 → 索寇要塞第二場 50 隻全滅。

## 提示詞（可直接貼給 `/goal`）

> 目標：做 #28 的 house rule，然後讓主線探針在它開著的情況下（另一條收據）靠交件升級，
> 走過索寇要塞交件與貧民窟 25 場；主台帳 #28、#22。
>
> 1. **選項。** `poolsave.State` 加 `HouseRules.CommissionExperience`（預設 false，跟存檔走）；
>    開新遊戲的隊伍選單加一個切換鍵，畫面上開著時要看得到（狀態列或標題列標「自訂規則」）。
>    不做全域設定檔——它是這一場戰役的規則，不是這台機器的偏好。
> 2. **發放點。** 市政廳獎賞那一場「空戰鬥」（`ecl3/8 9F27h CLEARMONSTERS → TREASURE → COMBAT`）
>    的戰後結算：`awardCombatExperience` 在怪物清單為空、且這一份 `TreasureRequest` 來自
>    市政廳（`ECL3/8`）時，把七欄貨幣換成金幣等值（`internal/treasure` 的換算：白金 ×5、
>    寶石／珠寶用估價表）當總額，照 spec 097 除以人數、套主屬性加成。關著時 0。
>    其他區的空戰鬥寶物（貧民窟的袋子、樓板下的箱子）**不算**——那是撿到的，不是委任。
> 3. **測試。** 開著／關著各一條，從 `Update()` 走職員交件；數值要對 spec 137 的獎賞表
>    （貧民窟 250 金＋50 白金＋1 珠寶 → 依估價換算）。
> 4. **探針第二條收據。** 複製主線探針成 `TestMainlineProbeHouseRuleCommissionExperience`
>    （或用旗標切），開 house rule；策略層補：任一人經驗過門檻就走去訓練所（ECL3/11
>    `PROGRAM 0`，spec 097 `trainMember`，付 1000 金）、有錢就回武具店換更好的甲（板甲、
>    盾）、神殿治療付得起才用。每段記等級、XP、金幣、最大一場、結果。原版規則那一條
>    探針保持不動，仍是 #5 的收據。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」；先跑最小重現**（交件一次、訓練一次），整條探針
>    只用來量結果，結果表接進 `docs/playtest/mainline-end-to-end.md`。不准加大 timeout。
> 6. **不可越線：** 不注入旗標／座標、不 forced-win、不改能力值或金幣、不動 engine／CoAB；
>    house rule 只在自己的開關後面，關著時所有既有測試與對拍數字不變（跑一次整包確認）。
>    動到玩家看得到的畫面（切換鍵、標記）就跑 `tools/package-release.sh` ＋
>    `tools/appimage-dos-parity.sh`，數字寫進 commit。
> 7. **台帳：** 每輪在 #28／#22 留言；選項＋測試＋文件（README「已知限制／自訂規則」一段）
>    完成後關 #28；探針在 house rule 下走到索寇交件＋貧民窟 25 場後在 #22 留言並回 #5。
>    關閉後同步 `docs/worklist.json` → `go run ./cmd/pool-worklist -mode render -write WORKLIST.md`
>    → `-mode verify` → `gh issue list --state open` 盤點。
> 8. **停止線：** 同一區、同一死因第三次；或發現要動 `internal/combat` 的判定；或要改原版
>    規則才過得去（house rule 之外）；或一條探針超過 60 秒——停下來回報。

## 已知風險與待決

- 寶石／珠寶的估價在原版是擲骰的（spec 115／`internal/treasure/appraise.go`），
  折算經驗值要定「用估價前的基準值」還是「估價後」；先用基準值並寫明。
- 訓練所升級的 HP 重算（spec 097 CONFORMED）與法師學新法術（DRAFT）——升到二級的法師
  要挑一條新法術，remake 讓玩家自己挑，探針要按得出來。
- house rule 開著時貧民窟的 24／33／34 那三場仍是門檻；二級隊伍打不打得過要量，不要猜。
