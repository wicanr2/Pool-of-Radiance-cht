# Spec 068：法術名稱表

狀態：CONFORMED（表的位置、步長、筆數、56 個名稱、六個分組與中譯對照）；
DRAFT（法術的等級／職業欄位是否另有資料、記憶與施展、效果與時效）。
日期：2026-09-03。

## 表的形狀

法術名稱表**編譯在 `START.EXE` 裡**，不像物品型別表那樣另存一個檔：

| 項目 | 值 |
|---|---|
| 檔案位移 | 41052（第一筆 `Bless`） |
| 步長 | 41 bytes |
| 筆數 | 56（最後一筆 `Restoration` 起於 43307） |
| 每筆 | 1 byte 長度 ＋ 名稱 ＋ 補零 |
| 編號 | **1-based**：遊戲用 1..56，1 是 `Bless`（見 spec 070） |

名稱之後的位元組**全是零**，所以職業與等級不在這張表裡。

overlay-22 以 `2883h + id × 41` 取名，而 `2883h + 41 = 28ACh` 就是第一筆，
所以遊戲自己用的編號是 1-based；本規格的 `index` 欄是 0-based 的表位置，
兩者差一，接線時用 `SpellByID` 換算（spec 070）。

## 分組是推出來的，但有獨立佐證

分組由條目順序推得，並與說明書下冊第六章逐條核對：六個分組的界線正好落在
說明書的 `LEVEL` 標題上，組內順序也逐條相同。

| 索引 | 職業 | 等級 | 筆數 | 首／末 |
|---|---|---:|---:|---|
| 0..7 | 神術 | 1 | 8 | Bless／Resist Cold |
| 8..20 | 巫術 | 1 | 13 | Burning Hands／Sleep |
| 21..27 | 神術 | 2 | 7 | Find Traps／Spiritual Hammer |
| 28..34 | 巫術 | 2 | 7 | Detect Invisibility／Strength |
| 35..43 | 神術 | 3 | 9 | Animate Dead／Bestow Curse |
| 44..55 | 巫術 | 3 | 12 | Blink／Restoration |

`Restoration` 是這一組裡唯一與 AD&D 分級對不上的（規則書是牧師第五級），
說明書的法術章也沒有它——它在神殿服務那一份選單裡。這一點照實記著，
不替它改組。

## 中譯來自說明書，比對以遊戲的拼法為準

56 條的中文名全部取自說明書下冊第六章（軟體世界的官方譯本）。

說明書把六個法術名拼錯了，因此比對要以**遊戲**的拼法為準，不能反過來：
`SPIRITOAL`（SPIRITUAL）、`KEY OF ENFEEBLEMENT`（RAY OF）、`BESTOW URSE`
（BESTOW CURSE）、`LIGHTNENG BOLT`（LIGHTNING）、`Stinking Clud`／`Stinking cioud`
（Stinking Cloud）、`PROTECTION FROM NORMAL MISSILF`（MISSILES）。
另有兩處是縮寫而非錯字：`PROTECT FROM GOOD`、`FIND TRAP`。

說明書沒收的兩條 `Protection From Evil／Good, 10' Radius`，說明書在巫術第三級
以 `PROTECTION FROM EVIL／GOOD` 列出並註明「類似 LEVEL 1 的…但威力…」，
即同一條；中譯沿用該處的「免受邪惡／善良傷害」並補上半徑。

## 閘門

內建的表與 `START.EXE` 逐字比對（`TestBuiltInSpellNamesMatchTheOriginalTable`）。
分開存是為了配中譯，但只要有一條抄錯，畫面上就會出現原版沒有的法術名，
而那看起來像翻譯問題，不像資料錯位。另有測試固定六個分組的筆數與首末，
以及「同一職業同一等級之內中譯不重名」。

## 玩家路徑

遊戲裡按 `K` 開法術一覽，`TAB` 換級別。**這是查閱用的畫面，不是施法**：
記憶與施展還沒接，畫面上也不假裝接好了。原版的法術書畫面另有版面
（說明書 p.51 有圖），等反組譯讀到再做。

驗收：`docs/screenshots/pool-remake-chinese-spells.png`。

## 下一步的已知線索

預設人物檔的 `.spc` 已由 spec 069 解出：它是角色的**效果串列**，不是記憶的
法術（持有它的人包括一名 8 級戰士與一名 9 級賊）。法術施展之後要把效果掛到
同一條串列上，所以那份規格是記憶與施展的前置。

## 不做

- 不替法術補等級／職業以外的資料。表裡沒有，就不從 AD&D 規則書搬。
- 不做記憶與施展。那要先讀 `.spc` 與 overlay 的施法路徑。
