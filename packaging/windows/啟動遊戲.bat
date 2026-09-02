@echo off
rem 從這個 .bat 所在的目錄啟動，讓遊戲找得到旁邊的原版 ZIP。
cd /d "%~dp0"
rem 有倚天字型就用中文介面；把 stdfont.15 放在同一層即可。
if exist "stdfont.15" (
  pool-game.exe -lang zh -eten-font stdfont.15 %*
) else (
  pool-game.exe %*
)
if errorlevel 1 pause
