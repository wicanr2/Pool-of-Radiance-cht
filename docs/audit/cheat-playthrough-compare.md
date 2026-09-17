# 兩邊作弊通關的逐段對照（#5）

由 `tools/cheat-playthrough-compare.py` 產生，別手改。來源：
`docs/audit/remake-cheat-playthrough.json`（`TestMainlineProbeCheatMenuToEnding`）與 `docs/audit/dosgolem-cheat-playthrough.json`（dosgolem `57454c4`，駕駛 `tools/dosgolem-cheat-playthrough.py`）。

對照三項（goal 第 8 步）：必經區塊（spec 137 的段落，比「兩邊都走過」與先後順序）、段末主線旗標、結局。
隨機遭遇、回合數、時刻、路過的其他區塊不列入；多走的區塊寫在備註（城區 ECL3/0 不列）。
remake 的段落以旗標切；原版以駕駛的段落收據為準。

| 段 | 內容 | 必經區塊 | 段末主線旗標 | 路線旗標 | 備註 |
|---|---|---|---|---|---|
| slums | 貧民窟 25 場 | 一致（ECL2/20） | 一致 | 相同 |  |
| handin-slums | 交貧民窟的件 | 一致（ECL3/8） | 一致 | 相同 | 原版另走 ECL2/20；原版穿牆 1 次 |
| podol | 波多廣場拍賣並交件 | 一致（ECL8/29 → ECL1/18 → ECL3/8） | 一致 | 相同 | remake 另走 ECL2/20；原版另走 ECL2/20；原版穿牆 3 次 |
| norris | 諾里斯並交件 | 一致（ECL8/29 → ECL3/8） | 一致 | 相同 | remake 另走 ECL2/20；原版另走 ECL2/20；原版穿牆 2 次 |
| sokal | 索寇要塞並交件 | 一致（ECL4/21 → ECL3/8）；出口不同：ECL4/21 之後 remake 進 ECL2/20／原版進 ECL3/0（#44） | 一致 | 相同 | remake 另走 ECL3/11、ECL2/20；原版穿牆 3 次 |
| ending | 東航線 → 城堡 → 覲見廳 | 一致（ECL8/27 → ECL7/26 → ECL1/18 → ECL2/9 → ECL5/3 → ECL5/5 → ECL5/7） | 一致 | 4A77 remake 60／原版 7F、4A78 remake 00／原版 03 | remake 另走 ECL3/11、ECL3/8；原版另走 ECL5/6；原版穿牆 18 次 |

## 結局

| 項 | remake | 原版 |
|---|---|---|
| `4ABA` | FE | FE |
| 結局後回到 | ECL3/0 | ECL3/0 |
| 結局頁 | 3 頁（spec 137 第 11 段） | 駕駛攔到 1 頁：「HIDING. NONE ATTACK.」 |

結局頁數不同（#45）。

沒開 issue 的主線不一致：0 段（共 6 段）；已開 issue 的差異：2 項。
