# Spec 134：原版的法術清單版面

狀態：READY（幾何量自原版畫面；標題的七個句尾、清單與底部資訊的位置）；
DRAFT（y 120..127 那一列的內容、清單超過一頁時怎麼捲）。
日期：2026-09-09。

## 輸入

原版畫面：dosgolem 走到「紮營 → MAGIC → MEMORIZE」的那一幀
（`workplace/dosgolem-ref-spells/57-m.idx`，320×200 的 EGA 色號陣列）。
鍵序見下面〈怎麼走到〉。

## 幾何（exact，量自色號陣列）

單位是 native 像素；remake 的邏輯畫布是它的兩倍。

| y | 內容 | x 起 |
|---|---|---|
| 1..6 | 上框繩索 | — |
| **8..14** | 標題 `HERO'S SPELLS IN SPELL BOOK` | 9 |
| 17..22 | 繩索橫條（標題與清單之間） | — |
| **40..46** | `1ST LEVEL` | 11 |
| **48..54** | 第一條法術 | 25 |
| … | 行距 **8** | 25 |
| **104..110** | 第八條法術 | 25 |
| 120..127 | 見下面〈那一列不是版面〉 | — |
| 129..134 | 繩索橫條（清單與資訊之間） | — |
| **144..150** | 角色名 | 9 |
| **152..158** | `CAN MEMORIZE:` | 9 |
| **160..166** | `CLERIC SPELLS: 3` | 41 |
| 185..190 | 下框繩索 | — |
| **192..198** | 指令列 `CHOOSE SPELL: MEMORIZE EXIT` | 9 |

字高 7、行距 8。**清單縮排 16 個像素**（`1ST LEVEL` 在 11、法術名在 25）。

指令列與冒險畫面同一條基線（native 192..198，spec 123），在繩索框**外面**。

## 標題的七個句尾

同一支介面（overlay-19 entry 12，spec 119）依 `[bp+10h]` 換句尾：

| 值 | 句尾 | 什麼時候 |
|---|---|---|
| 0 | `in Memory` | `C)AST`：已記憶的法術 |
| 1 | `in Spell Book` | `MEMORIZE`：法術書（上圖就是這一種）|
| 2 | `on Scroll` | 抄一張卷軸 |
| 3 | `on Scrolls` | 抄多張 |
| 4 | `to choose from` | 學新法術 |
| 5 | `to be memorized` | 已排入記憶佇列的 |

**版面是同一份**，只有標題那一行的句尾與指令列不同。所以拍得到 `MEMORIZE`
那一種就等於拍到了 `C)AST` 那一種。

## 那一列不是版面

y 120..127 那一列在畫面上是一串看不懂的白色字元（色號 15 占 262／320）。
**不要照著畫**：那是未初始化緩衝被顯示出來，dosgolem 上的內容與真機不會
一樣（同一個位置在別的畫面也出現過同樣的雜訊）。原版那裡實際上放什麼
還沒讀出來。

## 怎麼走到

```
建人類牧師 → 開始冒險 → 導覽走完 → E（紮營）→ M（MAGIC）→ M（MEMORIZE）
```

完整鍵序（`tools/dosgolem-reference.sh` 的 `POOL_DOSGOLEM_KEYS`）：

```
rep:9:Space,Return,Return,c,rep:5:End,Return,Return,Return,Return,Return,
Return,y,H,E,R,O,Return,k,e,y,a,a,e,b,rep:18:Return,e,m,m
```

`rep:5:End` 是把種族清單的反白從 `DWARF` 移到 `HUMAN`——**清單用 End／Home
移動，不是 Down／Up**（spec 133）。

**`C)AST` 那一種還沒拍到**：一級牧師沒有記憶法術，要先休息讓 `MEMORIZE`
生效，而貧民區休息一定會被城市守衛打斷
（`YOU ARE ROUSTED BY THE CITY WATCH…`）。那一段留給接上「休息被打斷」
之後再走。

## remake 現況

挑法術那一步已經照這一份改成整頁（`drawSpellPage`）：標題、一條橫線、縮排的
清單、再一條橫線、底下「還施得出來」，指令列在框外。挑人那一步維持在下方
小框——那一步是 remake 自己加的（原版對「目前角色」施法，spec 119）。

每一行的位置就是原版 native 座標乘二，由
`TestSpellPageMatchesTheOriginalRows` 逐行釘住。兩處是
`layout-reconstructed`：

- **標題那一行往下讓到 62**（原版 native 14 ＝ logical 28）。那個位置在
  remake 是框上緣自己的標題列，原版沒有那一行。
- **兩條橫線畫成一條線**，不是繩索圖塊。這一頁畫在面板上，再畫繩索會與外框
  的繩索疊成一團。

**remake 這一頁還沒有實拍**：截圖流程建的是戰士，按下去只會得到「沒有記憶
法術」。要拍它得先建施法職業、用 `K` 記一條、紮營休息讓它生效——那一段還沒
接進截圖腳本。
