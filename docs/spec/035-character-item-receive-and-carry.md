# Spec 035：角色接收物品、16 格上限與力量負重

狀態：CONFORMED（接收成功／OverLoaded、物品數、重量、力量容量、schema 3 與原子 Take）；DRAFT（已裝備欄位、
金錢重量、丟棄／交易、物品效果與戰鬥規則）。
日期：2026-09-01。

## 位址鏈與工具

- Spec 034 的 `overlay-05:0C23h` bytes `9A 2A 00 41 00` 呼叫 TPOV stub。runtime
  segment 不可用 overlay index 猜：START.EXE MZ header 是 `3B0h`，segment `41h`
  對應 control record file offset `7C0h`，即 overlay-06；stub `2Ah` 是 entry 2、
  code `0230h`。`docs/audit/ida-overlay06-item-receiver.json` 保存原始 bytes。
- `overlay-06:0230h..033Dh` 先呼叫 `0C9h:004Dh`。該位址在 START.EXE 是 `CD 3F`
  overlay stub；control segment `0C9h` 對應 overlay-19，stub `4Dh` 是 entry 9、
  code `274Fh`。`docs/audit/ida-start-c9-004d-overload.json` 保留 stub，
  `docs/audit/ida-overlay19-overload-check.json` 保存真正 handler。
- overlay-19 又呼叫 segment `10Ah` 的 stubs `43h/6Bh`；control record 對應
  overlay-25 entries 7／15、code `0BBEh`／`13F8h`，見
  `docs/audit/ida-overlay25-carry-weight.json`。
- 全部使用 IDA Pro 9.4；overlay 位址為 local offset、base 0，MZ stub 同時保存 runtime
  segment:offset 與 IDA linear address。沒有用推測改名取代原定位。

## 接收契約（exact）

1. overlay-25 `0BBEh` 沿角色 `+C8h` inventory 鏈，以 item `+2Ah/+2Ch` 為 next；
   重算角色 `+C7h` item count 與 `+102h` total load。
2. 每筆重量先取 item `+37h` word；若 `+39h > 0`，重量乘以該 byte。
3. overlay-19 `274Fh` 在加入前判斷：
   - 目前 `item count > 15`，或
   - `current load + new item load > carry adjustment + 1500`
   任一成立即 overload。
4. overload 時 overlay-06 顯示原始字串 `OverLoaded`，回傳 failure；Spec 034 的 loot
   節點因此不移除。成功時配置 63 bytes、完整複製 item、next 清零，附加到角色
   inventory chain，重算 derived data，再回傳 success。
5. 因檢查發生在加入前，現有 count `15` 仍可加入第 16 件；現有 `16` 才拒絕。

## 力量容量表（exact raw adjustment）

overlay-25 `13F8h` 先取得原版 strength table index，再回傳 adjustment：

| index | adjustment | 最終容量（+1500） |
| --- | ---: | ---: |
| 1..3 | -350 | 1150 |
| 4..5 | -250 | 1250 |
| 6..7 | -150 | 1350 |
| 8..11 | 0 | 1500 |
| 12..13 | 100 | 1600 |
| 14..15 | 200 | 1700 |
| 16 | 350 | 1850 |
| 17..21 | `500 + (index-17)×250` | 2000..3000 |
| 22..26 | `2000 + (index-22)×1000` | 3500..7500 |
| 27 | 7500 | 9000 |
| 28..30 | `9000 + (index-28)×3000` | 10500..16500 |

index 必須由共用 engine 已驗證的 `ability.StrengthIndex` 產生；非法 STR／exceptional
percentile 失敗即關閉，不能 clamp。

## Remake 與存檔契約

- 角色 inventory 每件保存 display name 與完整 63 raw bytes；raw bytes 才是未來欄位
  解碼的權威，名稱不能取代 record identity。
- 本規格把存檔升為 schema 3；現行已由 Spec 037 升為 schema 4。schema 2 仍以空
  inventory 確定性遷移，schema 1 先依既有 HP 遷移再得到空 inventory。讀取時驗證
  63 bytes、Pascal name 與顯示名稱一致。
- Take 必須是原子操作：容量檢查失敗不改角色、不移除 pending loot；成功才同時加入
  inventory 並移除一筆 pending loot。寫檔失敗要回滾兩側。
