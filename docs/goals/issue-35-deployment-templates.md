# Goal：陣型樣板從原版執行期讀出來對（GitHub #35；spec 061）

主台帳：[issue #35](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/35)。
上一個 goal 的結案位置：[`issue-32-deployment-and-pathing.md`](issue-32-deployment-and-pathing.md)
（#32／#34 關：部署 spec 061 CONFORMED、走位 spec 096 CONFORMED，同骰流 40/40）。

## 現況（2026-09-16）

- 樣板 `DS:43A2h`（2 組 × 4 陣型 × 6 列 × 11 欄）不在 `START.EXE` 檔內，是 overlay-10 `1A99h`
  開打前用 `DS:304h` 五組列範圍填的——這是**靜態讀出**（`1B93h..1C9Ah` 反組譯 ＋
  `ida-start-deployment-tables.json`）。remake `combat.FillDeploymentTemplates` 照它重建。
- 五筆 dosgolem 收據的擺放結果逐格相同，但那只踩到陣型 0 與各象限用到的那幾格；陣型 1..3、
  F=4 整塊、以及 `304h`／`43A2h` 執行期有沒有別的寫入端，都還是推論。
- 工具：`tools/dosgolem-drive-orc-home.py`（`-peek` 可加位址；`-trace-peek`／`-trace-call` 都在）、
  `cmd/pool-disp-scan`（掃 disp16 寫入端，要帶正對照）、`tools/ida.sh`。
- 順序：#31（諾里斯 THAC0，小）可先做，不衝突。

## 提示詞（可直接貼給 `/goal`）

> 目標：把陣型樣板從原版執行期讀出來，證明 remake 的 `FillDeploymentTemplates` 與原版填出來的
> 528 bytes 逐 byte 相同；主台帳 #35。
>
> 1. **先讀原版，不改 remake。** dosgolem 在四場開打那一幀（獸人家、衛兵攔截、驚動衛兵、哥布林）
>    `-peek ds:43A2:528`，同一幀帶 `ds:304:60` 當正對照（讀到的要等於 `ida-start-deployment-tables.json`
>    的位元組，否則是 DS 抓錯）。收據進 `docs/audit/dosgolem-deployment-templates-*.json`，含 generator、
>    dosgolem commit、原版 SHA-256、鍵序。
> 2. **追寫入端。** `304h` 與 `43A2h`（含 `43A2h..45A1h` 整段）用 `cmd/pool-disp-scan` 掃 36 顆 overlay ＋
>    `START.EXE`，正對照帶 `1A99h` 那幾條已知寫入；每一筆標位址、bytes、證據等級進 spec 061 的樣板一節。
>    有 `1A99h` 以外的寫入端就先讀它，再決定 remake 要不要改。
> 3. **對表。** remake 同狀態（同朝向、同距離、同人數）印 `FillDeploymentTemplates` 的 528 bytes，
>    測試逐 byte 對；差的標邊、陣型、列、欄。全對就把 spec 061 樣板一節從「靜態讀出」升 exact
>    （執行期對過），狀態行更新。
> 4. **陣型 1..3 的收據。** 找一場讓放置掉到陣型 1 以後（把一邊擠滿——dosgolem 可以先把隊伍擺到
>    夾道、或挑敵方 30 隻以上那一場看第一腿放不下的），對 `1609h` 換陣型的路徑；踩不到就寫明為什麼。
> 5. **不可越線：** 不改能力值／裝備、不注入旗標、不動 engine／CoAB；樣板資料**不搬進 game pack**
>    （它是執行期填的，spec 061 已說明）。動到畫面才跑對拍。
> 6. **每輪開跑前寫「這輪跟上輪差在哪」**；先一場對 528 bytes，再上四場。
> 7. **台帳：** 每輪在 #35 留言；spec 061 狀態行；收尾 `docs/worklist.json` → `go run ./cmd/pool-worklist
>    -mode render -write WORKLIST.md` → `-mode verify` → `gh issue list --state open`。
> 8. **停止線：** DS 抓不到（正對照對不上）連試三種鍵序仍失敗；`43A2h` 有 `1A99h` 以外的寫入端且
>    那一支跨兩顆 overlay 以上還讀不完；四場對完有差但差在 remake 沒實作的陣型（那是新 issue）。

## 已知風險與待決

- `43A2h` 那一段是未初始化資料段，開打前的殘值可能是上一場的；讀的時機要在 `1A99h` 跑完之後、
  第一個提示之前（開打那一幀就是）。
- 陣型 1..3 可能只有大場面才踩到；擠不到就用 dosgolem 收據證明「三場都在陣型 0 收完」。
