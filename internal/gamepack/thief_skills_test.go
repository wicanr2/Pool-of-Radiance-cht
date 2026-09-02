package gamepack_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 預設人物裡唯一的賊（chrdatd4，賊等級 9）那八格，要與規則書第 9 級賊
// 逐項對得上；其餘六名的八格必須全是 0。這一條同時釘住了順序與偏移。
func TestThiefSkillsMatchTheRulebookRow(t *testing.T) {
	record, err := os.ReadFile("../../workplace/oracle/dos/chrdatd4.sav")
	if err != nil {
		t.Skipf("original character records unavailable: %v", err)
	}
	skills, err := gamepack.ThiefSkillsFromRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	// 規則書第 9 級賊：80／77／65／78／63／30／97／45。
	// 記錄裡的三項各高 1..3，是敏捷與種族修正。
	for _, item := range []struct {
		index int
		want  uint8
		note  string
	}{
		{gamepack.ThiefSkillPickPockets, 80, "掏包"},
		{gamepack.ThiefSkillOpenLocks, 77, "開鎖"},
		{gamepack.ThiefSkillFindRemoveTraps, 65, "找／解陷阱"},
		{gamepack.ThiefSkillHearNoise, 30, "聽聲"},
		{gamepack.ThiefSkillReadLanguages, 45, "讀語言"},
	} {
		if skills[item.index] != item.want {
			t.Fatalf("%s 是 %d%%，規則書第 9 級是 %d%%", item.note, skills[item.index], item.want)
		}
	}
	if skills[gamepack.ThiefSkillClimbWalls] < 95 {
		t.Fatalf("爬牆 %d%%，第 9 級的賊應該接近 97%%", skills[gamepack.ThiefSkillClimbWalls])
	}
	for _, name := range []string{"chrdatd1", "chrdatd2", "chrdatd3", "chrdatd5", "chrdatd6", "chrdatd7"} {
		other, err := os.ReadFile("../../workplace/oracle/dos/" + name + ".sav")
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		got, err := gamepack.ThiefSkillsFromRecord(other)
		if err != nil {
			t.Fatal(err)
		}
		if got != (gamepack.ThiefSkills{}) {
			t.Fatalf("%s 不是賊卻有技能值 %v", name, got)
		}
	}
}
