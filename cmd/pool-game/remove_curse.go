package main

import (
	"fmt"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
)

// 除咒術的物品那一半（overlay-22 `2508h`，spec 098〈除咒術〉，issue #66）：
//
//	2516  0100h:006Bh(目標, 24h)            ; overlay-24 entry 15：身上有 24h 就摘掉、回 1
//	2520  回 1 → 印 "is un-cursed"（24E6h），結束
//	2547  否則沿物品串列（記錄 +C8h，下一件 +2Ah）：
//	2569    +36h != 0 → 清成 0、立旗標，迴圈在旗標立起時停     ; 只清一件
//	258D  旗標立著 → 印 "'s item is un-cursed"（24F3h）
//
// `+36h` 非 0 的物品卸不下來（overlay-19 `14FAh`，gamepack.ItemCursedOffset）。神殿的
// H）EAL 也交給同一支（internal/temple/remove_curse.go）。

const (
	// msgCastItemUncursed 是 `24F3h` 的 "'s item is un-cursed"。
	msgCastItemUncursed messageID = iota + 5000
)

func init() {
	for id, key := range map[messageID]string{
		msgCastItemUncursed: "ui.castItemUncursed",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// uncurseFirstItem 是 `2547h..258Bh`：物品串列裡第一件被詛咒的清掉，回傳有沒有清到。
// 神殿與法術同一支（temple.UncurseFirstItem）。
func uncurseFirstItem(member *poolsave.Character) bool {
	return temple.UncurseFirstItem(member)
}
