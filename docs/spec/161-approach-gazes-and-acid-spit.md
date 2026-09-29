# Spec 161：接近之前的凝視與噴酸（效果群組 0Eh 的 `53h`、`54h`、`79h`）

狀態：READY（三支處理常式逐條讀過位元組並實作，從 `Update()` 送鍵測試）。證據是位元組
（exact），另註者除外；沒有原版執行期收據逐次對過。日期：2026-09-27。主台帳：GitHub issue #82。
接手 spec 098〈吐息〉結尾「群組 0Eh 其餘三個碼還沒接」那一句；`58h` 吐息仍在 spec 098。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
  `START.EXE` `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`（MZ 初始 SS:SP = `0F92h:2000h`）。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 09 | `6b47e49d1a2e73427b9863560350965d6d5b43d3dd89df89d9b7f43a68409258` | — |
| 12 | `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e` | — |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 24 | `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714` | `0100h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |
| 31 | `64f1f7b8d8de27f5440391134f563eedf12e35e686d554de9af9afd35bc151df` | `0138h` |
| 32 | `efc22ba82144187c4bfc5deadb2bb10a868c13e45f2f7ca63f54d7387b22dfc8` | `013Dh` |

- 反組譯：`coab-go-test:20260729` 的 GNU objdump（`-D -b binary -m i8086 -M intel`）。far call 的
  segment 以 (`executable_file_offset` − stub offset − 3B0h) ÷ 16 對 manifest 反查成 overlay／entry。
- 處理常式位址來自 `docs/audit/dos-effect-handlers.json`：`53h` = overlay-12 entry 76（`1CC5h..1E87h`）、
  `54h` = entry 77（`1E87h..2011h`）、`79h` = entry 115（`2C32h..2D4Ch`）。三支都是 `retf 0Ah`
  （記錄 `[bp+0Ch]`、節點 `[bp+08h]`、模式 `[bp+06h]`）。
- 誰帶著：`docs/audit/monster-atlas.json`。BASILISK（MON2／MON5 block 26）帶 `7Fh 53h`、MEDUSA（MON5
  block 49）帶 `40h 7Fh 53h`、VAMPIRE（MON4 block 23／105）帶 `54h … 7Eh …`、AHNKHEG（MON6 block 65）
  帶 `50h 79h`；節點都是 `0000FF00`（`+4` 為 0，不跑收尾）。遭遇（`docs/audit/dos-ecl-scripted-fights.txt`）：
  曼多爾圖書館 ECL2 block 15 `A5A2h`（BASILISK）、ECL5 `9E3Dh`（MEDUSA）、瓦海登墳場 ECL4 block 10
  `B161h`／`B270h`（VAMPIRE）、ECL6 `9D1Bh`／`A817h`（AHNKHEG）。

## 派發

群組 0Eh 只有 overlay-09 entry 5 開場 `0B66h` 一個呼叫端（spec 112）。overlay-24 entry 3（`02E2h`）依
`53h 54h 58h 79h` 的順序各叫一次 `014Dh(記錄, 碼)`：`0177h` overlay-25 entry 27 找到最早的那個節點就
`02D9h` 以模式 0 叫 entry 1（`0000h`）分派——**每個碼只叫一次**，不管身上有幾個。處理常式的目標都是
自己 runtime `+0Ah`：entry 1 的迴圈在 `37B8h` 挑好（spec 096）之後才進 entry 5。

## `53h` 石化凝視（`1CC5h`）

```
1CCBh  c4 7e 0c / 26 c4 bd 08 01 / 26 c4 45 0a     6784h = 自己 runtime +0Ah（目標）
1CF0h  9a 34 06 bb 05 … 9a 84 00 0a 01            entry 20(自己, "gazes...", 0Ah, 0)（1CA5h）
1D03h  9a 98 00 0a 01                              overlay-25 entry 24(12h)（組四格圖，spec 166）
1D42h  9a 9d 00 0a 01                              overlay-25 entry 25(自己 X,Y, 目標 X,Y, 4, 2Dh)（動畫）
1D47h  b0 7f … 9a a7 00 0a 01 / 08 c0 / 75 03      overlay-25 entry 27(自己, 7Fh)：沒有 → 1E38h
1D65h  26 c4 85 c8 00                              走 6784h 的物品串列（記錄 +C8h，+2Ah 是下一個）
1D8Dh  26 80 7d 34 00 / 75 03                      +34h（裝備中）為 0 → 下一件
1D9Ah  26 80 7d 2f 76 / … 30 76 / … 31 76          +2Fh／+30h／+31h 有一格是 76h（"Mirror"）→
1DC7h    entry 20(目標, "reflects it!", 0Ch, 0)（1CAEh）、反向動畫、[bp-5] = 1、6784h = 自己
1E38h  b0 01 50 / b0 00 50 / 9a 43 00 00 01         overlay-24 entry 7(6784h, 1, 0)（類別 1，修正 0）
1E4Bh  08 c0 / 75 1e                               豁免過了 → 返回
1E57h  b0 07 … 1CBBh "is Stoned" … e8 ed e1         005Ah(6784h, 7, "is Stoned")
```

沒有距離與視線的判斷，也不叫 entry 34——凝視完照常進接近迴圈。`76h` 是 `START.EXE` 物品名稱字詞表
（`DS:10BBh + 字詞 × 15h`）的 "Mirror"（`workplace` 探針讀 `ReadDOSItemNameTable`，`76h` 前後是
"Medallion"／"Necklace"）。

`005Ah`（overlay-12，`retf 0Ah`）：`0073h` entry 20(記錄, 訊息, 0Ah, 0)；`008Ch..009Bh` 狀態已經是
7／6／8 就返回；否則 `00A5h` `+10Ch = 狀態`、`00ADh` `+10Dh = 0`、`00B6h` `+11Bh = 0`、`00C2h`
overlay-24 entry 13（`1004h`，摘 `DS:0C28h` 那十六個碼）、`00D0h` 群組 13、`00E6h` 還躺著就 overlay-32
entry 20。與中毒致死同一支（spec 153）。

## `54h` 魅惑凝視（`1E87h`）

```
1E8Dh  6784h = 自己 runtime +0Ah
1ED0h  9a 57 00 96 00 / 08 c0 / 75 03               overlay-13 entry 11（1087h）(自己, 目標)：0 → 返回
1F0Bh  9a 39 00 38 01 / 08 c0 / 75 03               overlay-31 entry 5（0419h）(自己 X,Y → 目標 X,Y，
                                                   預算 [bp-2])：0 → 返回
1F27h  entry 20(自己, "Gazes...", 0Ah, 0)（1E73h）；1F3Ah 組圖、1F79h 動畫（spec 166）
1F7Eh  c6 06 79 67 0a                              DS:6779h = 0Ah（魅惑人類）
1F91h..1FBFh                                        entry 20 的引數：(目標, 0Bh, 持續 0,
         (自己 +10Eh << 7) + 0Ch, 1, 1, overlay-24 entry 7(目標, 4, FEh), "is charmed")（1E7Ch）
1FD4h  9a 84 00 00 01                              overlay-24 entry 20（1656h）
1FE9h  9a a7 00 0a 01 / 08 c0 / 74 19               entry 27(目標, 0Bh)：找到 → 2006h entry 1 模式 0
```

`1087h`（`retf 8`）：候選是 nil 回 0、等於自己回 1，其餘派發群組 1（候選）與群組 0（自己）後
`677Ch` 為 0 才回 1。吸血鬼帶 `7Eh`，群組 0 的 `7Eh` 認的正是目標手上裝備中的 `76h`（鏡子）與
`98h`／`FCh`（spec 112），所以**拿著鏡子的人不會被凝視**。entry 20 見 spec 098〈模式 0Ah〉：群組 9
免疫 → 豁免成功而規則 1 → 印 "is Unaffected"；否則掛 `0Bh`（持續 0、有收尾）；`0Bh` 模式 0 是
overlay-12 entry 14 的倒戈（spec 112）。群組 9 的魔法抗性用 `010Ah:00D4h(DS:6779h)`：魅惑人類是
法師法術，讀 `DS:5CF0h`（吸血鬼）的 `+9Bh`。

### `0419h` 的預算是殘值（strong inference）

`1F02h` 推給 `0419h` 的預算變數是處理常式自己的 `[bp-2]`，`1E8Ah` `sub sp,14h` 之後沒有任何指令寫它。
那個位址上一次被寫是在 `37B8h` 最後一次 `1087h` 裡：entry 1（overlay-09）對 `37B8h` 推 10 bytes 引數、
對 entry 5 推 4 bytes，兩者的 bp 差 6；由此往下算，`54h` 的 `[bp-2]` 正好是 `1087h` → 群組 0 → `014Dh`
→ overlay-24 entry 1 的 `push bp`——存的是 `014Dh` 的 bp，一個堆疊位移。吸血鬼帶 `7Eh`，群組 0 一定
走到那一次 entry 1。`START.EXE` 的初始 SP 是 `2000h`，所以這個值落在幾千，上限 `預算 × 2 + 1`
遠大於盤面上最長的一條線（49 格斜走，成本 147）。結論：**只看地形擋不擋**。唯一的例外是 `37B8h`
第二輪放寬（`DS:6674h +6` 立著）而 `38E5h..38F4h` 跳過 `1087h` 的那一條，殘值改由 `0912h` 那一串留下，
沒有追。remake 用 `withinArea(自己, 目標, 0FFh)`，與殘值在這張盤面上結果相同。

## `79h` 噴酸（`2C32h`）

```
2C38h  6784h = 自己 runtime +0Ah
2C53h  b0 01 50 / b0 64 50 / 9a 48 00 00 01 / 3c 19 / 76 03    Roll(1, 100) > 25 → 返回
2C6Dh  9a c5 00 0a 01 / 3c 04 / 72 03                          overlay-25 entry 33（2591h）距離 >= 4 → 返回
2C7Fh  9a ca 00 0a 01                                           overlay-25 entry 34（266Dh）結束行動
2C97h  entry 20(自己, "Spits Acid", 0Ah, 1)（2C27h）；2CAAh 以槽 17h 組圖、2CE9h 動畫（1, 1Eh；spec 166）
2CF6h  b0 08 50 / b0 04 50 / 9a 4d 00 00 01                    overlay-24 entry 9(8, 4)（8d4，骰數記進 677Ah）
2D02h  b0 02 50 … b0 03 50 / b0 00 50 / 9a 43 00 00 01          entry 7(目標, 3, 0)
2D19h  9a 7f 00 00 01                                           overlay-24 entry 19(目標, 傷害, 規則 2, 豁免)
2D2Dh  b0 79 … 9a 2a 00 00 01                                   entry 2(自己, 79h, 這個節點)
2D41h  b0 50 / 31 c0 / 31 d2 … 9a 2a 00 00 01                   entry 2(自己, 50h, nil)：沿串列摘第一個 50h
```

Pascal 由左往右求引數，所以 8d4 在豁免之前擲。`2591h`（`retf 8`）在 `25B0h` 把 `DS:6674h` 的 `+6`
立起來（不判地形），以預算 FFh、不指定朝向叫 `0912h`，找到目標那一筆取成本除以二（`264Bh`）；
找不到時迴圈停在最後一筆（`2628h` 的 `jae`）。處理常式沒有寫 `DS:6779h`／`6777h`（群組 6 讀到殘值，
停止線）。所以 AHNKHEG **每一場只噴一次**，噴完連群組 2 的 `50h` 一起失去。

## remake 的對應

- 規則常數：`internal/gamepack/approach_effects.go`（`ApproachEffectCodes`、`MirrorReadied`、
  `AcidSpitAfterUse`、各碼的類別／修正／門檻）。
- 盤面：`cmd/pool-game/approach_effects.go`（`foeGazeStone`、`foeGazeCharm`、`foeAcidSpit`、
  `entry33Distance`、`statusOffBoard` = `005Ah`）；派發 `cmd/pool-game/breath.go` 的 `foeApproachEffects`，
  由 `foeTurn` 在 `foeChooseGear` 之後呼叫。
- 凝視把目標石化或魅惑之後，原版接近迴圈 `0C7Bh` 作廢目標、搆不到人時 `0D5Dh` 叫 `37B8h` 重挑；
  remake 的接近迴圈不逐步重挑，所以凝視過後目標不再有效就在 `foeApproachEffects` 當場重挑一次（挑不到
  就收工）。規則相同，差在擲骰的先後：原版只在搆不到人時才重挑。
- 石化的隊員狀態 7、生命 0、下場；下一場開打照 `+10Dh = 0` 只擺屍體（`partyMemberDown`）。
- 測試（全部從 `Update()` 按 ENTER）：`TestBasiliskGazeTurnsTheTargetToStone`、
  `TestMirrorReflectsTheGazeOnlyWhenReadied`、`TestGazeDoesNotEndTheAction`、
  `TestVampireGazeCharmsUnlessTheTargetHoldsAMirror`、`TestVampireGazeNeedsAClearLineAndAFailedSave`、
  `TestAnkhegSpitsAcidOnceWithinThreeSquares`、`TestAnkhegHoldsTheAcidOutOfReachOrOnAHighRoll`、
  `TestStonedMemberIsDeployedAsACorpse`。變異檢查：拿掉反射、石化、視線、距離門檻、摘節點各一次都有測試變紅；
  拿掉 `54h` 裡的 `1087h` 不會變紅——`foeTurn` 開頭的 `37B8h`（`foeTarget`）已經先過一次 `1087h`，
  在 remake 這一條是重複的閘，照原版留著。

## 與原版不同或未追的（停止線）

| 項目 | 分類 | 理由 |
|---|---|---|
| 凝視後重挑目標的擲骰先後 | 未知 | 規則相同；remake 的接近迴圈本來就不逐步叫 `37B8h`（spec 096） |
| `54h` 在 `37B8h` 第二輪放寬時的預算殘值 | 未知 | 只有第一輪一個候選都挑不到才走得到，殘值由 `0912h` 留下，沒有追 |
| `54h` 留下 `DS:6779h = 0Ah`、`79h` 不寫 `6779h`／`6777h` | 未知 | 讀它們的是法術傷害那一路的群組 6（鏡影、魔法抗性、火／冷免疫）；下一次施法會覆寫，remake 當 0 |
| 三支的聲音、動畫與訊息列位 | 呈現 | 只影響畫面 |
