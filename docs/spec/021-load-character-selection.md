# Spec 021：`0Ah LOAD CHARACTER` 選定角色與 Pool 記憶體投影

狀態：CONFORMED（低七位選擇、找不到保留前次角色、inline projector 與本輪三欄投影）；
DRAFT（高位 legacy redraw 副作用、NPC 加入流程與完整 285-byte record 投影）  
日期：2026-08-31

## 範圍

本規格只解除初始地圖 sweep 最後 12 筆 fail-closed error。它不把完整 DOS 角色 record
硬塞進共用引擎，也不宣稱 remake 已有 NPC 加入／離隊；只實作目前玩家路徑實際讀到的
active-character 姓名、`+100h` 探針與 `+0B8h` control／morale byte。

## 輸入與位址空間

- DOS ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- IDA Pro 9.4、image `ida-pro-9.4-idapython:locked-v1`、metapc 16-bit raw binary；
  handler 位址為 overlay-local file offset、base 0。
- 非破壞性匯出：`docs/audit/ida-overlay03-op0a-op20-op24.json`；`02E3h..03B0h`
  保留原始 bytes、operand、工具版本與輸入雜湊。
- ECL3/block 0 mapping base `9900h`；真實呼叫端為 `AF59h`。

## 原版 handler（exact）

Pool dispatcher `334Ah..3353h` 將 opcode `0Ah` 路由到 `02E3h`。handler：

1. 解析一個 numeric operand；`value & 7Fh` 是零起算角色索引，`value & 80h` 另存為
   legacy restore／redraw 旗標。
2. 由 party head far pointer `DS:5CF4h` 開始；索引每增加一，沿角色 record
   `ES:[DI+104h]` 的 next far pointer 前進一次。
3. 目的 pointer 非零才覆寫 active far pointer `DS:5CF0h/5CF2h`；鏈提早結束時不清空，
   而是保留先前 active character。
4. 高位未設即返回。高位已設時仍須 `DS:8298h`、`DS:8299h` 都非零才進入
   `036Dh..03A8h` 的 legacy buffer／redraw 路徑；本切片不猜這個前端副作用。

## 真實 consumer（exact／strong inference）

ECL3/block 0 `AF53h..AF88h` 把 `9802h` 初始化為 0，重複：

- `AF59h LOAD CHARACTER [9802h]`；
- `AF5Dh COMPARE [6C00h], 7Fh`；
- `AF68h` 比較 string-memory `[6B00h]` 與 `[9890h]`；
- 未命中就增加 `9802h`，最多掃到 8。

命中姓名後 `AF89h` 比較 `[6BB8h]` 與 `80h`。`6B00h` 是 active-character 字串視窗、
`6C00h = 6B00h + 100h`、`6BB8h = 6B00h + 0B8h` 的幾何是 exact；姓名與 morale
語意由同一控制流及 CoAB 已驗證的第二作品 selected-character projection 交叉支持，
在 Pool 先標 `strong inference`。現行玩家建立的隊員不是 NPC，morale 投影為 0；日後
NPC record 必須由自己的 RE 規格提供原始 byte，不可用姓名猜。

## Typed 行為

1. 共用 VM 解析 numeric operand，產生作品中立的
   `{value, index: value & 0x7F, high_bit}` selection，並立即呼叫 title projector；
   projector 缺席或回錯時失敗即關閉。
2. projector 可在下一條 ECL 執行前更新 VM 的 numeric／string memory；共用引擎不含
   `6B00h`、party 長度、角色姓名或 NPC 規則。
3. Pool projector 對有效索引寫姓名到 `Strings[6B00h]`、`Memory[6C00h]=1`、
   `Memory[6BB8h]=ControlMorale`。無效索引完全不改這三格，符合原版保留前次 active
   pointer；session 尚未選過角色時維持零值／空字串。
4. projector 捕捉的是建立 session 當下的 immutable party snapshot；Clone 可共享函式，
   但不得共享可變 memory／strings。正常遊戲由 `save.State.Party` 建 snapshot；隔離 sweep
   的空 party 是明示測試條件。
5. 高位保留在 result request 供前端日後處理；本切片不模擬 `8298h/8299h` 的 DOS
   redraw buffer，也不把它冒稱完成。
6. `AF68h` 直接暴露共用 VM 尚未泛化的 string-memory `COMPARE`。只要任一 operand
   是 ECL text form（`80h/81h`），兩邊都以 `ecl.TextValue` 解析並設定六個字典序旗標；
   一邊文字、一邊 numeric 必須失敗即關閉。這與 CoAB 已驗證 runtime 的同 opcode
   契約一致，不是 Pool 特例。

## 驗收

- engine synthetic：低七位／高位解碼；有效 projector inline 改值並影響下一條
  COMPARE；projector 缺席失敗即關閉；Clone 的 memory／strings 隔離；字串 COMPARE
  的相等／不等與 mixed-type fail-closed。
- Pool synthetic：索引命中更新三欄；越界保留上次選擇；空 party 不產生假角色。
- 正常遊戲建立 session 時帶入目前 party snapshot；讀檔後開始冒險同樣適用。
- 重生 `docs/audit/pool-initial-cell-sweep.json`：12 筆 `0Ah` error 歸零；新 boundary
  依實際結果分類，不用 passthrough 或放寬期望掩蓋。
- engine／Pool 全測試與 CoAB ECL／game 核心回歸通過。

## 停止線

本切片只做玩家路徑需要的三欄。完整 CHA byte projection、NPC morale 來源、high-bit
renderer restore 只有在玩家功能實際依賴時另開窄規格；不得為追求整份 record 逐 byte
一致而延伸成無限 RE。

## 實作收據

- 共用引擎 commit `0a028e0956c1` 加入 title-neutral `CharacterSelection`、inline
  projector 與 string-memory `COMPARE`；projector 缺席及 mixed operand type 都維持
  fail-closed，完整引擎測試通過。
- Pool 正式鎖定
  `v0.0.0-20260831132230-0a028e0956c1`，正常遊戲以 `save.State.Party` 建 immutable
  snapshot；隔離 sweep 明示使用空 party。
- sweep 由 856 `EXIT`／156 event／12 error 收斂為 856 `EXIT`／168 event／0 error。
  新增 12 筆全是三個 City Hall 格、四朝向的原始文字
  `YOU ARE OUTSIDE THE CITY HALL...`，不是空事件或測試期望放寬。
- Pool 全套測試已在斷網 Docker／Xvfb 以正式 pseudo-version 通過。
