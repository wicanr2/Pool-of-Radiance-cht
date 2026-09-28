package main

import (
	"flag"
	"os"
	"path/filepath"
)

// 本機完整版（full-local）把原版 ZIP 與倚天字型放在執行檔旁邊的 `data/`，
// 讓玩家點兩下就能開。可散布的 patch 包沒有這個目錄：原版資料與字型都沒有
// 公開散布權（README〈下載〉），那一版照舊要自己用 -zip／-eten-font 指定。

// executableDir 是執行檔所在的目錄；測試換成暫存目錄。
var executableDir = func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// bundledDataDir 是執行檔旁邊的 `data/`；不存在就回空字串。
func bundledDataDir() string {
	base := executableDir()
	if base == "" {
		return ""
	}
	dir := filepath.Join(base, "data")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

// bundledFile 是 `data/` 裡的一個檔；沒有就回空字串。
func bundledFile(name string) string {
	dir := bundledDataDir()
	if dir == "" {
		return ""
	}
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return ""
}

// flagWasSet 說命令列有沒有明確給這個旗標；給了就一律照玩家的意思，不換成包裡的檔。
func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// resolveBundledInputs 在沒有指定 -zip／-eten-font 時改用包裡的檔。
// ZIP 的順序是：-zip 指定的、目前目錄的預設檔名、執行檔旁的 data/。
// 字型只在 -eten-font 沒給時才從 data/ 帶；-lang auto 因此在完整版預設是中文。
func resolveBundledInputs(zipPath, etenFont *string) {
	if !flagWasSet("zip") {
		if _, err := os.Stat(*zipPath); err != nil {
			if bundled := bundledFile(filepath.Base(*zipPath)); bundled != "" {
				*zipPath = bundled
			}
		}
	}
	if !flagWasSet("eten-font") && *etenFont == "" {
		*etenFont = bundledFile("stdfont.15")
	}
}
