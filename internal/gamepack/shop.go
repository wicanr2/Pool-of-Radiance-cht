package gamepack

// 商店。原版把商店做成與墓園戰利品同一條服務邊界：
// `CLEARMONSTERS → TREASURE → SAVE → COMBAT`，差別在 TREASURE 請求的內容。
//
// 墓園帶著七種貨幣的數量與 item block `33h`；四家商店的七個數量全是零，
// item block 分別是 `34h`..`37h`，而且會另外寫三個旗標
//（`6E6Ch=1`、`6EF6h=1`、`6E6Dh=16`）。所以「這是商店還是戰利品」不看
// item block，看那三個旗標——只看 block 會把新加的戰利品當成商店。
//
// 進貨清單就是 `ITEM3.DAX` 的那幾個 block，格式與戰利品完全相同（spec 033）。
// 交叉核對：block `35h` 的 `Long Sword` 記錄 `+2Eh` 是 `24h`，而物品型別表
// 第 `24h` 筆是 1d8／1d12，正是 AD&D 的長劍；價格欄 15 也與規則書相同。
const (
	// ShopServiceKindAddress 是 ECL 記憶體裡的服務種類欄位。
	ShopServiceKindAddress = 0x6E6D
	// ShopServiceKind 是商店的值。
	ShopServiceKind = 16
	// ShopServiceEnabledAddress 與 ShopServicePartyAddress 是同一批寫下的兩個
	// 旗標，語意尚未閉合，但四家商店都寫 1，所以一起當成邊界條件的一部分。
	ShopServiceEnabledAddress = 0x6E6C
	ShopServicePartyAddress   = 0x6EF6
)

// itemPriceOffset 是物品記錄裡的價格（金幣），word。
// 對照原書價目：長劍 15、板甲 400、匕首 2、鑲嵌皮甲 15、鏈甲 75，都相符。
const itemPriceOffset = 0x3a

// Price 是這件物品的售價，單位是金幣。
//
// 有幾筆的價格是 0（箭、弩矢、標槍、投石索），那是原始資料就寫 0，不是讀錯欄位
// ——同一個欄位在同一個 block 的其他 55 筆都對得上原書價目。
func (r TreasureItemRecord) Price() uint16 {
	return uint16(r.Raw[itemPriceOffset]) | uint16(r.Raw[itemPriceOffset+1])<<8
}
