# Spec 142：怪物的物品串列——從哪裡載、誰用、戰後怎麼交出來

狀態：READY（載入鏈、第二隻起的順序、entry 3 用物品、戰後錢與物品併入戰利品、@6DE3，
實作與測試）；DRAFT（戰利品價值折算經驗值、怪物武器／護甲重算、NPC 分錢、戰後標題字樣）。
日期：2026-09-26。主台帳：GitHub issue #76（#71 留下）。

## 輸入與位址空間

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
  `MON1ITM.DAX..MON8ITM.DAX`（八個都在）。
- overlay（`workplace/ovr/`，對 `docs/audit/dos-ovr-manifest.json`），位址一律是
  overlay-local file offset、base 0，反組譯是 `coab-go-test:20260729` 的
  `objdump -D -b binary -m i8086 -M intel`：

| overlay | SHA-256 |
|---|---|
| 03 | `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f` |
| 05 | `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16` |
| 07 | `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae` |
| 17 | `f92fed1bcf009daa8638ba0ab1f8f9a680c0ad3b2b43b799faa9383b02f7d1e4` |
| 19 | `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9` |

far call 由 `segment = (executable_file_offset − stub_offset − 3B0h) ÷ 16` 對 manifest 反查：
`0045h:003Eh` = overlay-07 entry 6（`0423h`）、`00BEh:0048h` = overlay-17 entry 8（`0E90h`）、
`0100h:0025h` = overlay-24 entry 1（`0000h`）、`010Ah:0025h` = overlay-25 entry 1。

## 物品從哪裡來（exact）

`0Bh LOAD MONSTER` 的處理常式是 overlay-03 `044Dh`（spec 077 的表）。它在 `04A6h` 呼叫
overlay-07 entry 6，後者 GetMem(11Dh) 之後在 `0451h`（`9A 48 00 BE 00`）呼叫 overlay-17
entry 8 `0E90h`，回來再把記錄的 `+C8h`（物品）與 `+7Fh`（效果）兩個串列頭交還給 `044Dh`。

`0E90h` 用同一個 block 編號（`[bp+0Ah]`）依序讀三個檔，檔名都是
`"MON" + Str(DS:52D4h) + 後綴 + ".dax"`：

```
0EB7  "MON"(0E64h) + Str(52D4h) + "CHA"(0E68h)  → 0F28h Move 11Dh bytes 進記錄
0F3B  清 +C8h、+7Fh、+104h、+108h 四個遠指標
1051  "MON" + Str(52D4h) + "SPC"(0E88h)         → 9 bytes 一節，鏈在 +7Fh、下一節 +5
112E  "MON" + Str(52D4h) + "ITM"(0E8Ch)         → 3Fh bytes 一節，鏈在 +C8h、下一節 +2Ah
116F  block 長度 0 → 整段跳過（沒有物品）
1178..11F7  offset 從 0：GetMem(3Fh)、Move(block+offset, 節點, 3Fh)、
            清節點 +2Ah／+2Ch（`11BAh`）、offset += 3Fh，offset < 長度就再接一節
```

所以怪物的物品串列就是 `MONnITM.DAX` 同一個 block，一件 63 bytes，順序照檔案，
內容原封不動（名字欄是空的，`+2Eh` 起才有資料）。八個檔對到 MONnCHA 的 172 筆，
共 301 件，每個 block 都是 3Fh 的整數倍（`TestRealMonsterItemChainsAreWholeRecords`）。

`ADD NPC`（overlay-17 entry 9 `1216h`）也是 `1244h` 呼叫 `0E90h`，所以 NPC 帶的物品
是同一份來源。整個 GAME.OVR 只有 overlay-17 有 `ITM` 這個後綴；`0ACDh` 那一支讀的是
存檔的 `.sav`／`.cha`／`.itm`／`.spc`（`0AA5h` 起的字串），不是怪物。

### 同一條 `LOAD MONSTER` 的第二隻起（exact）

`044Dh` 對數量 2 以上逐隻 GetMem(11Dh)、Move 記錄，再沿第一隻的串列逐件複製
（`0629h..06E9h`）：`+C8h` 還是空的就放成頭（`0645h..0682h`），否則 GetMem(3Fh) 放成新
的頭、**舊的頭接到新節點的 `+2Ah`**（`0684h..06D5h`）。所以第二隻起物品順序是反的。
效果串列（`06FAh` 起）是同一個形狀。

## 怪物怎麼用物品（exact）

overlay-09 entry 3（`03E3h`，spec 096〈entry 3〉）與 overlay-19 entry 8（`1A86h`）不分
隊員或怪物，都從記錄 `+C8h` 掃。remake：`foe_items.go` 的 `foeItemBearer` 對怪物回
`tacticalState.FoeItems`，記帳走 `foeItemSpender`（`SpendAIItemUse`，用完從串列摘掉）。

- **施法者等級**：`1BBDh` 立 `DS:6CB3h` 之後才進 overlay-22 entry 5（`0C14h`），
  overlay-25 `26F8h` 看到它就把牧師／法師法術當 6 級、物品效果（參數表 `+0 == 2`）
  12 級（spec 098）。`0C14h` 在 `0C1Ah` 取 `DS:5CF0h` 當施法者，等級交給各處理常式經
  `26F8h` 取，`DS:6CB3h` 立著時不讀記錄的 `+96h`／`+9Bh`（`2739h..2752h`），所以
  等級與拿著物品的是誰無關，exact。
- **`+3Eh`**：overlay-19 `14CBh`（穿上／卸下）`14D4h` 比 `+3Eh > 7Fh`
  （`26 80 7D 3E 7F 77`），成立時卸下在 `152Bh` 以 `(模式 1, 物品, 記錄, +3Eh)` 呼叫
  overlay-24 entry 1（效果碼分派，spec 112），穿上在 `1653h` 以模式 0 呼叫。所以
  **`+3Eh >= 80h` 是物品穿戴時掛上的效果碼**（分派 exact；每一碼的處理常式另讀）。資料
  裡量得到：火焰抗性戒指 81h、食人魔之力手套 83h、位移斗篷 85h；卷軸的 `+3Eh` 小於 80h。
  entry 3 的 `04ADh` 跳過它們：那一類的 `+3Dh` 不是拿來放的法術。
- 真的會被 entry 3 挑到、而且處理常式已讀的有 12 件（LEVEL 3 MU 的杖、AL-HYAM DAZID
  的閃電杖、LEVEL 5 CLERIC 的項鍊、COMMANDANT 的標槍、YARASH 的杖等）。

## 戰後錢與物品（exact）

overlay-05 主流程 `14CAh` 先呼叫 `04ADh`；`04ADh` 在 `05E0h` 看 `DS:82A0h`（有隊員站著）
才呼叫 entry 2（`0000h`）。`0000h` 沿 `DS:5CF4h` 的串列（隊員在前、怪物照載入順序）：

```
0068  +10Eh != 1（不是敵方）          → 下一個
0079  +10Ch == 3（逃掉）              → 下一個   ; 26 80 BD 0C 01 03，經驗值也跳過
0084  DS:439Ch = 1
0089..00BE  i = 0..6：DS:[6752h + i×4] += word +88h + i×2    ; 七種錢進公款
00C0..00F4  經驗值（spec 097）
00F7  [4937h]+5C6h == 1 → 物品整段跳過                         ; C4 3E 37 49 26 83 BD C6 05 01
0106  沿 +C8h 逐件：
0111    word +3Ah > 0 → 收
011B    否則：已收 8 件 → 不收；否則 Roll(1, 10) <= 3 → 收（`012Ah`、`012Fh`）
0136    收：件數 +1；overlay-25 entry 1 把名稱重組寫回 +0（spec 067）；
        GetMem(3Fh)、Move；+34h（穿戴中）= 0；插在 DS:676Eh 串列頭
```

- `+3Ah`（word）量得到的非 0 值只在魔法物品上（杖 30000／35000、項鍊 1200），一般武器
  護甲是 0。行為照碼（exact）；「它是價值」是 strong inference。
- `[4937h]+5C6h` 是 ECL @6DE3（class 1：`2A00h + 6DE3h × 2 ≡ 5C6h`）。整個 ECL 只有
  ECL3 block 0 `A764h` 一條 `SAVE 1 → @6DE3`，接在一場隨機編成的戰鬥前面（`A777h` 起
  四條 `LOAD MONSTER`）。主流程在 `15FBh`（`26 89 85 C6 05`）把它清回 0。
- 逃掉的判準與經驗值同一道；投降（4）與倒下的照收。
- 物品插在串列頭，所以最後收的排第一；這個串列就是 spec 034 戰利品選單的那一條。

remake：`monster_loot.go` 的 `collectMonsterLoot` 照上面收，`openMonsterLoot` 把錢加進
`PooledMoney`、物品交給既有的戰利品選單（spec 034），離開選單時（`exitTreasure`）才續跑
戰後腳本。怪物那一側什麼都沒交出來時不開選單，照舊直接續跑。

## 還沒接（DRAFT）

| 項目 | 位址 | 等級 | 為什麼沒接 |
|---|---|---|---|
| 戰利品折算經驗值 | overlay-05 `0224h..0306h`：公款七欄依幣值換算、`DS:676Eh` 串列上（`DS:5CF8h` 之前）每件 `+32h × 400` 加進經驗總額 | exact（碼）；`5CF8h` 的語意 unknown | 改動每一場的經驗值，主線檢查點會動；另開 issue |
| 怪物的武器、護甲重算 | overlay-25 entry 7（`0BBEh`，spec 063／065／080） | exact（碼） | 同上，會動命中、傷害、AC 與射程（`tacticalState.AttackRange`）；另開 issue |
| NPC 分錢 | overlay-05 `1295h`：`+84h > 7Fh` 的 NPC 依 `+85h & 7` 先拿走份額 | exact（碼） | 不在 #76 範圍 |
| 戰後標題 | overlay-05 `08E0h`：打過怪（`439Ch`）印 `The party has won.`（`0836h`），只有沒打怪才印 `The party has found treasure!`（`085Dh`） | exact | remake 的戰利品選單沿用 `found treasure` 那一句；`0E85h` 選單畫面上留哪一句沒有對拍 |
| 沒有戰利品時選單開不開 | overlay-05 `0E85h` | unknown | remake 只在有東西時開 |
| 同一條 `LOAD MONSTER` 第二隻起的**效果**串列反序 | overlay-03 `06FAh` 起 | exact（碼） | 效果那一側不在 #76 範圍 |

## 測試

- `internal/gamepack/monster_items_test.go`：八個檔 172 筆記錄、301 件、步長 3Fh、下一節
  指標清 0；MON2 block 1 五件的型別與數量、block 94 的杖；長度不整除失敗即關閉。
- `cmd/pool-game/monster_loot_test.go`（全部從 `Update()` 送鍵）：LEVEL 3 MU 在自己的回合
  用杖（次數 67 → 66）；兩隻狗頭人首領與法師打完，第二隻逃掉，錢與物品只算投降與倒下的、
  第二隻的物品順序是反的、杖排第一；Take → Items 交給隊員之後存得進存檔；@6DE3 == 1 時只收錢。
