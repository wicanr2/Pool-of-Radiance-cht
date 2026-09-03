# Spec 100：`DS:6DD5h` — 擋住整個世界的那一個變數

狀態：CONFORMED（兩個使用點、它們各自通往哪裡、以及「沒有任何 ECL 寫它」）；
OPEN（誰寫它、寫的是什麼值）。日期：2026-09-03。

## 一句話

貧民窟與索寇要塞的回程船**都掛在同一個變數上**：`DS:6DD5h` 不是 0 才會發生。
remake 從來沒有寫過它，所以兩條路都是死的——這才是「其餘 25 個 ECL block
走不到」的直接原因。

## 十八個使用點

`DS:6DD5h` 不是三、四個地方的特例，是**通用的離開機制**：29 個 ECL 區塊裡
有 18 個在入口 0 讀它（`tools/go.sh run ./cmd/pool-ecl-memory-audit
-addresses 6DD5`）。

| archive/block | 讀的位址 |
|---|---|
| ecl1/18 | `99B9` |
| ecl1/24 | `992E` |
| ecl2/9 | `9C68` |
| ecl2/15 | `99ED` |
| ecl2/20 | `9934` |
| ecl3/0 | `993A` |
| ecl3/14 | `9934` |
| ecl4/2 | `995B` |
| ecl4/10 | `9920` |
| ecl4/21 | `9918` |
| ecl5/3 | `9A89` |
| ecl5/4 | `9AA0` |
| ecl5/5 | `9A89` |
| ecl5/6 | `9B5A` |
| ecl6/1 | `992E` |
| ecl6/28 | `9934` |
| ecl7/17 | `9DE7` |
| ecl8/29 | `9918` |

下面三段是逐條讀過的例子。**它們不是全部**——先前這一節寫「兩個使用點」，
是因為當時只追過三個區塊。

**城區（`ecl3` block 0，入口 0 也就是每回合那一支）**：

```
993a  COMPARE DS:6DD5h, 0
9940  IF =
9941  GOTO 9965h            ; 是 0 就整段跳過
9945  SAVE 1, DS:49EBh
994b  CALL C01Eh
994f  SAVE 255, DS:6DC9h
9955  LOAD FILES FFh,FFh,7Fh
995c  SAVE 2, DS:6E12h      ; archive 2
9962  NEWECL 20             ; ← 貧民窟（ECL2 block 20，spec 043）
```

**索寇要塞（`ecl4` block 21，入口 0）**：

```
9914  GOSUB AE42h
9918  COMPARE DS:6DD5h, 0
991e  IF =
991f  GOTO 9998h            ; 是 0 就整段跳過
9923  PRINTCLEAR "'DO YOU WANT TO TAKE A BOAT BACK TO PHLAN?'"
9945  GOSUB AE32h           ; YES / NO
9963  PRINTCLEAR "'YOU BOARD A BOAT.'"
9977  SAVE 15, C04Bh / 1, C04Ch / 3, C04Dh   ; 回到碼頭 (15,1) 面向 3
9989  SAVE 3, DS:6E12h
998f  NEWECL 0
```

兩邊的形狀一模一樣：**站在邊上往外走**（城區的西門、要塞的碼頭），
`DS:6DD5h` 非零，ECL 才把隊伍送去下一張圖。

**貧民窟（`ecl2` block 20，入口 0）**：

```
9934  COMPARE DS:6DD5h, 0
993a  IF =
993b  GOTO 996Bh            ; 是 0 就整段跳過
993f  SAVE 0, DS:49FBh
9945  CALL C01Eh
9949  SAVE 255, DS:6DC9h
994f  COMPARE C04Dh, 1      ; ← 朝向
9956  SAVE 3, DS:6E12h      ; 面向 1（東）→ archive 3
995d  NEWECL 0              ;              → 回城區
9961  SAVE 8, DS:6E12h      ; 其他朝向 → archive 8
9968  NEWECL 29
```

**這一段是最強的一條證據**：`DS:6DD5h` 非零之後，**用朝向決定去哪一邊**。
只有「隊伍正要走出這張圖的哪一邊」才需要這樣分。

## 誰寫它：還沒找到

- **沒有任何 ECL 區塊寫它。** 29 個有文字的 block 裡，26 個靜態解得開，
  只有上面兩條 `COMPARE`，沒有 `SAVE`。（三個解不開的是 ecl5/7、ecl7/17、
  ecl7/22，都不是每回合入口。）
- **執行檔裡沒有任何一處直接定址它。** `START.EXE` 與 38 份 overlay 全掃
  兩位元組字面 `D5 6D`：**零筆**。
  這個「零」有做正對照才算數：ECL 位址不會自動出現在執行檔裡（`6E12h`、
  `4AA7h`、`4A21h`、`6E82h` 全掃也都是零筆——那些只有 ECL 自己讀寫）。
  但**引擎會碰的那些就找得到**：`6D45h` 16 筆、`4944h` 14 筆、`49C9h` 10 筆、
  `49EBh` 8 筆。`6DD5h` 只有 ECL 讀、沒有 ECL 寫，卻又不出現在執行檔裡，
  所以它是被索引或指標寫進去的（像 `es:[di+n]`）。
- 附近那一段（`6DC1h`、`6DC7h`、`6DC9h`、`6DCBh`）也被 ECL 讀寫，
  像是引擎與 ECL 之間的一組共用變數。

## 語意：「隊伍正要走出這張圖」（strong inference）

三條各自獨立的證據指同一件事：

1. **位置**：三個使用點都在地圖邊上（城區西門、要塞碼頭、貧民窟邊界），
   而且都通往「離開這一區」。
2. **朝向**：貧民窟那一支在 `DS:6DD5h` 非零之後用 `C04Dh`（朝向）挑鄰居
   ——面向東回城區，其他方向去 archive 8 的 block 29。**只有「往哪一邊
   走出去」需要這個分支。**
3. **行為**：在 remake 裡做一次一次性實驗——移動前，若這一步會讓座標
   越界就把 `DS:6DD5h` 設成 1，其餘設 0——遊戲立刻照著它自己的文字走：

   ```
   block 21（索寇要塞）→ 0（城區）→ 20（貧民窟）→ 29 → 18 → 26
   ```

   不設它的話這幾條路一條都不會發生。實驗碼沒有留下（見下一節的兩個
   後續問題），但這個行為是可重現的。

牆的查詢本身會把座標夾回 0..15（overlay-30 `0358h`，spec 099），
所以繞回之後那一格看起來是合法的——引擎另外記下「這一步本來會走出去」
才說得通。

**還沒證實的是寫入點。** 要閉合得找到誰寫它：從移動服務往下讀，
看它在套用位移前後改了哪些 DS 變數。

## `LOAD FILES` 的 handler 讀完了（overlay-03 `0D80h`）

順著這條線把 `21h` 那一支逐條讀完，兩件事釘死了：

```
dc3  cmpb $0FFh, 第一欄 ; je → 跳過地圖載入
dc9  cmpb $07Fh, 第一欄 ; je → 跳過
dcf  cmpw $0, es:[di+1CCh] ; 隊伍狀態不允許也跳過
de4  mov es:[di+18Ah], ax   ; party +18Ah ← 第一欄
ded  lcall 0131h:0052h      ; ← GEO loader，**只收第一欄**
dfd  cmpb $0FFh, 第三欄 ; je → 結束；不是 FFh 才走另一個 loader
```

- **第二欄整支沒有 consumer**（與 spec 043 一致）。
- **GEO loader（overlay-30 `10EAh`）用 `DS:52D4h` 組檔名**，而 `NEWECL`
  的 handler（overlay-03 `0CDDh`）組 ECL 檔名用的**也是 `DS:52D4h`**。
  所以 ECL 與 GEO 共用同一個 archive 編號，remake 目前的模型（一個
  `spawn.Map.Archive`）是對的。

## 接上去之前要先修的兩件事

那次實驗也暴露了兩個目前擋著的問題：

1. **`NEWECL` 換區之後地圖沒有跟著換。** 實驗中的軌跡顯示 archive 一直在變
   （GEO3 → GEO2 → GEO8 → GEO1 → GEO7），但**`BlockID` 一路停在 21**——
   `a.spawn.Map.BlockID` 只有 `LOAD FILES` 會改，而貧民窟那一支
   （`ecl2` block 20 入口 0）**整段沒有 `LOAD FILES`**。所以換區之後跑的是
   舊地圖的地形，之後的分支全是垃圾。
2. 於是很快就撞上 `Pool LOAD FILES requested absent GEO7 block 5`：
   GEO7 只有 block 17／22／23／26，沒有 5。那個 5 是拿舊地圖的地形算出來的。

**先修 1 再接 `DS:6DD5h`**，否則接上去只是換一種壞法。

## 已經接上（2026-09-03）

`setMapExitFlag`（移動前判斷這一步會不會越界）與 `applyMapExitCommit`
（`CALL C01Eh`：依朝向走一格、邊界繞回）兩支都接在 `moveInitialDungeonForward`
與 `applyCellECLResult` 上，整包測試綠。

量到的世界：

| | 接之前 | 接之後 |
|---|---:|---:|
| 走得到的地圖 | 2 | **3**（訓練所那一區、**貧民窟**、索寇要塞）|
| 走得到的 ECL block | 4 | **5**（0、8、11、**20**、21）|

貧民窟（GEO2 block 20）是主線的第二站，攻略說的「拿到第一份委任之後往城門
走、跨過去」現在真的走得過去。

先前試接時報過「12 張圖、8 個 block」，那個數字是虛的：當時 `LOAD FILES`
還在追會落後的 archive，`spawn.Map` 會生出 GEO1/21 這種不存在的組合，
而且旗標沒清乾淨會一路換區換不停，把途中掃到的 block 都算進去。
編號查圖（spec 043）與 `CALL C01Eh` 的收尾都接上之後，**3 張圖 5 個 block
才是實際走得到的**。

## 接起來的過程

接法就是兩件事：跑格子入口 0 之前，若這一步會讓座標越界就把 `DS:6DD5h`
設成 1；看到 `2Dh CALL C01Eh` 就把它清掉，順便把那一步走掉。

第一次試接時擋著的兩件事（後來都解掉了）：

- **八個既有測試會紅**。起點 (0,4) 就在西邊界上，往西走一步現在會觸發
  離開事件而不是移動，`TestNormalKeysReachTheFirstDungeonStep` 那一類
  逐鍵重現的測試全部對不上。那些測試對過原版的是別的段落（spec 022 的
  驗收是 Sune → City Hall），西邊那一步到底該不該離開**沒有對過原版**。
- ~~走出去之後撞上 `LOAD FILES 5,5,0` 的硬失敗~~ —— **已解，見 spec 043**：
  區塊編號在八個 GEO 檔裡全域唯一，所以拿編號查地圖就對了，不必追那個會
  落後的 archive 值。這條已經落地（`GeometryCatalog.MapByBlock`），
  接不接 `DS:6DD5h` 都成立。

### 攻略把語意確認了

外部佐證：菲蘭分成幾區，**區與區之間靠邊界上的城門相接，走過去就到隔壁區**；
拿到第一份委任之後就是「往城門走、跨過去」進貧民窟。
來源：<https://the-spoiler.com/RPG/SSI/pool.of.radiance.1/Pool%20of%20Radiance/css/Pool%20of%20Radiance_1.htm>、
<http://crpgaddict.blogspot.com/2021/08/the-foundations-of-phlan-revisiting.html>。
這與三個使用點在做的事完全一致。

### 接上去之後會走到哪：貧民窟

隊伍從起點 (0,4) 往西走一步，落在 **GEO2 block 20 的 (14,4)**——
那正是攻略說的貧民窟，而且落點就是「西邊界的另一側」。
`TestPassiveCombatTerminates` 也跟著恢復綠燈。

再往下一個沒解的東西是**貧民窟的隨機遭遇**：`ecl2` block 20 `B1B0h` 起的
`LOAD MONSTER` 三個運算元都是記憶體參照（`DS:6E80h`、`DS:9808h`、
`DS:6E7Bh`、`DS:6E7Ah`），而那幾個變數在 remake 這邊是 0，於是排出
「怪物 0、數量 0」然後失敗即關閉。填那幾個值的地方還沒讀。

### 既有測試怎麼帶上來（已完成）

九個逐鍵重現的測試靠固定亂數種子亂走到某一場架。世界一變大，同一個種子就
走去別的地方，有幾個還會卡在新走得到的區域裡的選單。這不是遊戲壞了，
是測試的路線要重新確立。

處理方式**不是重挑種子**，而是把那些測試**留在起始區裡**：它們量的是戰鬥、
記憶法術、裝備，不是地理。`forwardWouldLeaveTheArea` 在按前進之前先問這一步
會不會走出這一區，會的話改成轉向。另外補上兩個新走得到的畫面的處理：
地圖上的隊伍管理（按 B 回地圖）與寶物選單（走到 Exit 再確定，而 Exit 後面
那句 Yes／No 要答 Yes）。

只有探索用的那一條（`TestDirectedExplorationReachesMaps`）刻意不設限——
它量的就是走得到多少。

照這個方向走了兩輪。第一輪剩三個卡在**同一個新問題**上：逐格走會先碰到另
一場架，而那一場久久分不出勝負——完全不還手的隊伍打到第 3629 回合，敵方
一直是「ATTACK MISSED」。那不是測試的問題，是**那一場敵人打不死人**：
六隻怪物的傷害骰讀錯格，每一擊都是 0 點（見 `internal/gamepack/monster.go`
的註解與 WORKLIST）。修掉之後第二輪全部綠。

最後改的形狀：**逐格走，不依賴種子**（`walkThisAreaUntil`），
並補上新走得到的畫面的處理（隊伍管理按 B、商店按 ESC、選單有 Exit 就走過去、
清光敵人之後那句「還要不要繼續打」要答 N）。幾個原本釘死「是哪一場架」的
斷言改成釘管線本身——第一場架現在是索寇要塞的毒蛙，那是路線決定的，
不是規則決定的。

### `+1CCh` 那道閘門查過了，不是答案

`LOAD FILES` 的 handler 在載地圖前會看 party `+1CCh`（`0DCFh`），為 0 就跳過。
本來以為那是出口。**全掃過所有寫入點**：整個遊戲只有一處寫它——overlay-07
`02D2h` 的 `movw es:[di+1CCh], 1`——而且只寫 1。所以它在正常遊玩時一直是 1，
那道閘門不會擋下 `LOAD FILES 5`。**這條排除掉了。**

所以 `LOAD FILES 5,5,0` 在原版一定是在 archive 5 的狀態下跑的，
問題出在「怎麼走到 block 26 的」——需要的是原版當 oracle，不是再多讀一段碼。

## `CALL C01Eh` 讀清楚了：那一步是 ECL 自己走的

`2Dh CALL` 的處理常式在 overlay-03 `3026h`：取一個運算元的值，減 `7FFFh`，
再拿差去比一串選擇子。`C01Eh` 那一支在 `30FAh`，轉呼叫 overlay-07 `1A17h`：

```
1a1a  al = DS:6A0Dh                 ; 朝向
1a1f  0（北）→ Y > 0 就 Y--，否則 Y = 15
1a38  2（東）→ X < 15 就 X++，否則 X = 0
1a51  4（南）→ Y < 15 就 Y++，否則 Y = 0
1a6a  6（西）→ X > 0 就 X--，否則 X = 15
1a81  重算地形（`DS:6A0Fh`）與牆（`DS:6A0Eh`）的暫存
```

**就是「把這一步走掉，而且在邊界繞回去」。** 三個讀 `DS:6DD5h` 的分支後面都
緊跟著它——所以走出這一區的那一步是 ECL 自己叫這一支完成的，不是引擎默默做
的。這一支落在 `applyMapExitCommit`。

順帶：`CALL 2C90h` 是同一支常式的另一個選擇子（`3055h`），只重算地形與牆的
暫存，不移動。

選擇子要看**運算元本身的字**，不是它指到的值：共用 VM 會把那個運算元當成
記憶體參照解出來（實測拿到 0），原版比的是字本身。

### 清旗標要清在哪：`applyCellECLResult`

旗標只對「這一步」有效，所以要有地方把它清掉。位置只有一個對：
**掃 `RunUntilEvent` 回傳的事件串，看到 `2Dh CALL C01Eh` 就清**
（`applyCellECLResult` 的第一件事）。兩個看起來合理的位置都不行：

- `RunInitialSessionCellEntry` 回來之後清 → **來不及**：`NEWECL` 的換區與
  新區塊的入口 0 都在同一次呼叫裡跑完（`eclvm/session.go` 的 `switchTo`
  之後是 `continue`，不會返回），新區塊的入口 0 還是看到 1。
- `applyTransitionResource` 進來時清 → `2Dh` 不走這一支，一次都沒進來。

## 這一條與 spec 099 的關係

099 說得沒錯：碼頭的其他航線被 `DS:4AA7h` 擋著，而那個旗標由索寇要塞寫。
`TestSokalKeepOpensTheOtherBoatRoutes` 已經證明**那一段 remake 推得動**
（`4A21=255`、`4AA7=254`）。但 099 漏掉了這一條：**要走回碼頭去用那些航線，
得先能離開要塞**，而離開要塞掛在 `DS:6DD5h` 上。099 裡「地圖沒有走出邊界
這回事」那句話只證明了一件事：**牆的查詢會把座標夾回去**。走出邊界這回事
存在，只是由 ECL 自己讀 `DS:6DD5h`、自己叫 `CALL C01Eh` 完成。
