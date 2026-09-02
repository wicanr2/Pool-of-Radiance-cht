package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 錢按比例減少，七種貨幣都算。
func TestRobMoneyScalesEveryCurrency(t *testing.T) {
	money := [gamepack.MoneyCurrencies]uint16{100, 55, 1, 0, 999, 7, 250}
	got := gamepack.RobMoney(money, 30)
	want := [gamepack.MoneyCurrencies]uint16{70, 38, 0, 0, 699, 4, 175}
	if got != want {
		t.Fatalf("得到 %v，預期 %v", got, want)
	}
	// 0% 不動，100% 拿光。
	if gamepack.RobMoney(money, 0) != money {
		t.Fatal("0% 不該動到錢")
	}
	if empty := gamepack.RobMoney(money, 100); empty != ([gamepack.MoneyCurrencies]uint16{}) {
		t.Fatalf("100%% 之後還剩 %v", empty)
	}
}

// 重的物品把成功率壓低，而且**壓下去不會回復**——原版是就地改參數，
// 所以背包裡先出現的一件盔甲會讓後面所有東西都比較難被偷。
func TestRobChanceKeepsFallingAcrossItems(t *testing.T) {
	chance := 95
	// 一件超過 255 的重物：扣 90。
	chance = gamepack.RobChanceAfterWeight(chance, 300)
	if chance != 5 {
		t.Fatalf("重物之後成功率 %d，預期 5", chance)
	}
	// 再一件中等重量：扣 50，壓到地板 0。
	chance = gamepack.RobChanceAfterWeight(chance, 30)
	if chance != 0 {
		t.Fatalf("再一件之後成功率 %d，預期 0", chance)
	}
	// 輕的東西不扣。
	if got := gamepack.RobChanceAfterWeight(40, 10); got != 40 {
		t.Fatalf("輕物把成功率改成 %d", got)
	}
	// 界線是「大於」，不是「大於等於」。
	if got := gamepack.RobChanceAfterWeight(80, gamepack.RobMediumWeight); got != 80 {
		t.Fatalf("重量剛好 %d 不該扣，卻變成 %d", gamepack.RobMediumWeight, got)
	}
}

// 擲骰小於等於成功率才被偷走。
func TestRobTakesItemUsesLessThanOrEqual(t *testing.T) {
	if !gamepack.RobTakesItem(30, 30) {
		t.Fatal("剛好等於成功率應該被偷走")
	}
	if gamepack.RobTakesItem(31, 30) {
		t.Fatal("超過成功率不該被偷走")
	}
	if gamepack.RobTakesItem(1, 0) {
		t.Fatal("成功率 0 不該偷得走任何東西")
	}
}
