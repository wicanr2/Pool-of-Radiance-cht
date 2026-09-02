# Pool of Radiance remake 工作歷程

本檔按日期追加已完成工作的短收據；目前真相見 `CONTEXT.md`，未完成工作與驗收條件
見 `WORKLIST.md`，逆向證據見 `docs/spec/`、`docs/re/` 與 `docs/audit/`。

## 2026-09-02：README 現況與截圖收據更新

- 目標：讓專案入口反映第一場 Slums 遭遇、Spec 048～053 與剩餘工作，重新驗證 README
  操作畫面，並登記下一輪 Journal 全文翻譯。
- 變更：修正 DOS ZIP 檔案數與 ECL VM 過期斷言；加入有明確分母的 20～25% 保守估計、
  Slums 戰鬥前 staging 能力與限制；`WORKLIST.md` 新增《軟體世界》說明書 Journal 翻譯
  的來源、術語及零遺漏驗收條件。
- 截圖：Docker／Xvfb 從真實 Ebitengine 視窗逐鍵重拍姓名輸入到 Rolf 導覽七張圖；輸出
  與版控 PNG 逐 byte 相同。新增 `docs/audit/remake-screenshot-manifest.json` 保存來源提交、
  日期、960×600 尺寸、SHA-256 與「正常 remake 玩家路徑、非 DOS parity」標籤。
- 驗證：七張 PNG 雜湊／尺寸／唯一性與 README 本機連結全部通過；`git diff --check`
  通過。嘗試由單元測試離屏 image 輸出戰鬥 staging 時，Ebitengine 因正式 command queue
  尚未建立而拒絕；已完整撤回該測試變更與暫存輸出，未加入 production direct-entry。
- 未完成：正常玩家路徑尚未抵達可操作戰術畫面，因此沒有戰鬥 runtime 截圖；下一輪先
  完成 Journal 翻譯 corpus，再進行 UI 接線。
- 提交／推送：本輪文件、manifest 與工作歷程提交至 `main`，並推送 `origin/main`。
- Docker 清理：所有一次性 Xvfb／Go／檢查容器皆以 `--rm` 結束，無專案容器殘留。
