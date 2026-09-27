package temple

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 除咒（issue #66）。神殿的 H）EAL 與牧師的法術最後走同一支 overlay-22 entry 9（`2508h`）：
//
//	overlay-04 entry 9（`081Fh`）神殿那一側：
//	  0845  沿物品串列（+C8h，下一件 +2Ah）找 +36h != 0 的    ; 有就是「有毛病」
//	  0888  沒有才問 010Ah:00A7h(記錄, 24h)                   ; 身上有降咒
//	  0891  兩樣都沒有 → "is not cursed" 並問要不要照做（預設 Y）
//	  08C5  00BFh 收 3500（0DACh），付了 → DS:6B89h = 這個人、00E2h:004Dh（overlay-22 entry 9）
//	overlay-22 `2508h`：
//	  2516  0100h:006Bh(目標, 24h)：有就摘掉最早的那一個、印 "is un-cursed"，結束
//	  2547  否則沿物品串列清掉第一件 +36h != 0 的（只清一件），印 "'s item is un-cursed"
//
// `+36h` 非 0 的物品卸不下來（overlay-19 `14FAh`，gamepack.ItemCursedOffset）。

// bestowCurseEffectCode 是降咒術掛的 `24h`（參數表 `+0Ah`，spec 098）。
const bestowCurseEffectCode = 0x24

// HasCursedItem 是 `0845h..0870h`：身上有沒有 `+36h` 非 0 的物品。
func HasCursedItem(character poolsave.Character) bool {
	for _, item := range character.Inventory {
		if len(item.Raw) > gamepack.ItemCursedOffset && item.Raw[gamepack.ItemCursedOffset] != 0 {
			return true
		}
	}
	return false
}

// UncurseFirstItem 是 `2547h..258Bh`：物品串列裡第一件被詛咒的清掉，回傳有沒有清到。
func UncurseFirstItem(character *poolsave.Character) bool {
	if character == nil {
		return false
	}
	for index := range character.Inventory {
		raw := character.Inventory[index].Raw
		if len(raw) <= gamepack.ItemCursedOffset || raw[gamepack.ItemCursedOffset] == 0 {
			continue
		}
		raw[gamepack.ItemCursedOffset] = 0
		return true
	}
	return false
}

// RemoveCurse 是 `2508h` 整支：先解 `24h`（最早掛上的那一個），沒有才清一件物品。
// 回傳解掉的是效果還是物品；兩個都是 false 代表什麼也沒發生。
func RemoveCurse(character *poolsave.Character) (effect, item bool) {
	if character == nil {
		return false, false
	}
	for index, node := range character.Effects {
		if node.Code == bestowCurseEffectCode {
			character.Effects = append(character.Effects[:index:index], character.Effects[index+1:]...)
			return true, false
		}
	}
	return false, UncurseFirstItem(character)
}
