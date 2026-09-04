# 原版走到第一人稱畫面的對拍基準

由 `tools/capture-dos-adventure.sh` 在一次性 Docker 裡重生（DOSBox ＋ Xvfb）。
鍵序寫在腳本的預設 `POOL_KEYS` 裡，重跑就會得到同一批畫面。

| 檔案 | 走到哪 | SHA-256 |
|---|---|---|
| `00-party-menu-empty.png` | 密碼過後的人物管理選擇項，**隊伍是空的** | `375804341398328372b497b8322fe39c87d0c5e53fcde9a84ad01ccf75d8f3c5` |
| `01-add-character.png` | `A)DD CHARACTER TO PARTY` 的名單頁 | `84f907b8324b32bf4b658e51f89180bb0d8d1142188ade312bef6c38aca61cec` |
| `02-first-person-start.png` | `B)EGIN ADVENTURING` 之後的第一人稱畫面 | `48b8a8e8c89011dd4ac2e85f4dbdb30930a5f681f3b320974d34196a89e2efc5` |
| `03-rolf-approach.png` | Rolf 初次 APPROACH 的圖像與第一頁台詞 | `a95eb80b15997bc9df5f8bbc27dc9ad8727c58ef1178379f559cc9f3e7265c00` |

## 逐張讀到的東西

**`00`**：畫面上只有四項——`CREATE NEW CHARACTER`、`ADD CHARACTER TO PARTY`、
`LOAD SAVED GAME`、`EXIT TO DOS`，底部 `CHOOSE A FUNCTION`。這是 spec 008
「隊伍是空的時候只剩那四項」的**第二份證據**（第一份是說明書 p.9 的文字）。
每一列的首字母另外用一種顏色標出來，與「以第一個字選擇之」相符。

**`01`**：底部是 `ADD A CHARACTER: ADD EXIT`。名單上只有剛建好的 HERO——
出貨的 `chrdat*` 檔**不在名單上**，因為 `CHARLIST.TXT` 是空的。

**`02`**：狀態列是 `15, 1 W 00:00`——位置 (15,1)、朝向 **W**、時鐘 00:00。
朝向與 spec 010 的「facing 6」在共用 engine 的 0/2/4/6 制下換算相符
（6 ÷ 2 = 3 ＝ 西，spec 076）。右邊是隊伍面板 `NAME / AC / HP`。
畫面內容：上半青色天空、下半藍色水面、棕色木棧道往遠處收窄、兩側棕色柱子，
棧道中央站著一個人形。

**`03`**：Rolf 的半身像在左框，台詞在下框，底部 `PRESS <ENTER>/<RETURN> TO
CONTINUE`。

## 與 remake 現況的差距（2026-09-05 第一次對拍）

`02` 與 `docs/screenshots/pool-remake-initial-first-person.png` **不一致**：

| | 原版 | remake |
|---|---|---|
| 天空 | 上半青色 | 全藍 |
| 地面 | 棕色棧道，往遠處收窄 | 灰色 |
| 兩側 | 藍色水面加棕色柱子 | 藍色格子加紅／粉紅柱子 |
| 棧道上的人形 | 有 | 沒有 |
| 右半版面 | 隊伍面板（`NAME / AC / HP`）與狀態列 | 除錯文字 |

所以「背景與同狀態 DOS 畫面」這一項現在的狀態是**已經對拍、不一致**，
不是「還沒對拍」。差異清單就是接下來要逐條收掉的東西。
