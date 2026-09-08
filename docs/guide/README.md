# 遊戲內攻略

遊戲裡按 `F3` 會開這張地圖的攻略：平面圖上標出地點，右邊列出離隊伍最近的
幾個。**預設只顯示走過的格子**；按 `V` 切完整攻略，第一次只出警告，
再按一次才攤開。

## 座標從哪來（這一條是硬規則）

**一律出自原始資料，不出自任何攻略。**

- 座標：原始 GEO 的地形碼（`cmd/pool-world-graph` 的 `cell_terrain`）配上
  [spec 102](../spec/102-city-location-dispatch.md) 的索引表。城區每一格
  「這裡有什麼」是 `terrain & 0x7F` 當索引查一張 `ON GOTO` 表，所以哪一格是
  哪一間店可以直接從原始資料讀出來。
- 名字：spec 102，而那一份的名字又是**原版腳本自己印出來的第一句話**。
- 地圖名：[spec 124](../spec/124-map-names-from-the-original.md)，判準是
  「先印一句話、緊接著 `NEWECL`」的同一條控制流配對。

**不可以把第三方攻略的座標抄進來。** 抄錯一格的症狀是「測試綠、玩家走不到」
——game pack 的宣告會蓋過原始資料，兩者在報表上分不出來（CoAB `CLAUDE.md`
的通則第 4 條）。`internal/guide` 的
`TestEveryGuidePointIsOnAWalkableCell` 就是這條的閘門，它直接回對原始 GEO。

## 第三方攻略只當檢查清單

[`docs/reference/walkthrough-notes-softworld-001.md`](../reference/walkthrough-notes-softworld-001.md)
整理了《軟體世界》創刊號那篇的區域事件數量。它的用途是**抓自己的盤點是不是
漏了一整塊**（「這一區該有幾個地點」），不是座標來源，敘述文字一律改寫。

## 目前建了哪些

| 地圖 | 名字 | 點數 |
|---|---|---:|
| `3/00` | 費蘭的文明區 | 48 |

其餘地圖還沒建點；打開時會說「這張地圖還沒有建過攻略點」，不會畫一張空地圖。
擴充的方式與驗收條件記在 [`WORKLIST.md`](../../WORKLIST.md)。

## 資料格式

`internal/guide/{zh-TW,en}.json`，schema `pool-guide-maps/1`：

```json
{"schema": "pool-guide-maps/1",
 "maps": {"3/00": {"title": "費蘭的文明區", "source": "docs/spec/124-...md",
   "points": [{"x": 0, "y": 4, "label": "城門",
               "summary": "往未開拓區；開場就站在這一格。",
               "source": "docs/spec/102-city-location-dispatch.md"}]}}}
```

每一個點都要有 `source`，指向 repo 裡建立那個座標的文件；兩個語言的點數與
座標要一致（`TestBothCataloguesCoverTheSameCells`）。
