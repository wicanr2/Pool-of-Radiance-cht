# 規格索引

> 這份由 `cmd/pool-doc-index` 產生，**手改會在下一次執行時被蓋掉**。
> 對應關係的主鍵是 spec 編號——程式碼註解裡的 `spec NNN` 就是那條線，
> 這份只是把它反過來收攏，所以改了註解重跑一次就對了。

135 份規格，其中 30 份還沒有任何檔案的註解指回它、59 份沒有測試提到它；
另有 6 份實作在共用 engine（`eclvm`），不在這個 repo。
這些數字是**盤點用的**：沒有反向引用不代表沒實作，只代表那條線還沒接起來。

## 規格

| # | 標題 | 狀態 | 實作 | 測試 |
|---|---|---|---|---|
| [001](001-dos-title-picture.md) | DOS TITLE DAX 圖像 | CONFORMED | `cmd/export-title/main.go` | — |
| [002](002-dos-ecl-code-base-and-parser-gap.md) | DOS ECL 位址基準與 parser 缺口 | DRAFT | `cmd/pool-ecl-audit/main.go`、`cmd/pool-ecl-frontier/main.go`、`cmd/pool-text-inventory/main.go` 等 5 個 | — |
| [003](003-dos-character-creation-flow.md) | DOS 建角流程 | READY | `cmd/pool-game/dos_export.go`、`cmd/pool-game/icon_menu.go`、`cmd/pool-game/main.go` 等 4 個 | `cmd/pool-game/icon_menu_test.go`、`cmd/pool-game/main_test.go`、`internal/gamepack/weapon_stats_test.go` |
| [004](004-dos-character-roll-fields.md) | DOS 角色擲值與持久欄位 | READY | — | `internal/gamepack/monster_test.go` |
| [005](005-first-remake-executable.md) | 第一支 remake executable | CONFORMED | `cmd/pool-game/main.go` | — |
| [006](006-dos-portrait-archives.md) | DOS 建角 portrait archive 形狀 | READY | — | — |
| [007](007-dos-combat-icons.md) | DOS 戰鬥圖示 | READY | `cmd/pool-game/icon_menu.go` | — |
| [008](008-character-library-and-party-menu.md) | 建角完成、角色庫與 Party Creation Menu | CONFORMED＋READY | `cmd/pool-game/party_menu.go`、`cmd/pool-game/training.go`、`cmd/pool-game/training_gate.go` 等 4 個 | `cmd/pool-game/party_menu_test.go` |
| [009](009-dos-geo-map-inventory.md) | DOS GEO 地圖盤點與 Phlan 入口 | CONFORMED＋READY＋DRAFT | `internal/gamepack/geometry.go` | — |
| [010](010-dos-initial-rolf-event.md) | DOS 初始 Rolf 導覽事件 | READY＋DRAFT | `internal/gamepack/intro.go` | — |
| [011](011-dos-rolf-guided-tour.md) | DOS Rolf 34-step 導覽與結束交接 | READY＋DRAFT | `internal/gamepack/intro.go` | — |
| [012](012-dos-cardinal-coordinate-wrapper.md) | DOS cardinal coordinate wrapper 與 401Fh dispatch | READY＋DRAFT | — | — |
| [013](013-shared-ecl-vm-rolf-path.md) | 共用 ECL VM 與 Rolf 真實 bytecode 路徑 | CONFORMED＋DRAFT | 共用 engine | — |
| [014](014-initial-map-basic-geo-walk.md) | 第一張地圖基本 GEO walk | DRAFT | — | — |
| [015](015-initial-map-cell-lifecycle.md) | 第一張地圖 cell lifecycle 接線 | CONFORMED | `cmd/pool-game/main.go` | — |
| [016](016-initial-search-location-and-sune.md) | 初始地圖 SearchLocation 與 Sune 第一個事件 | CONFORMED | `cmd/pool-game/command_bar.go` | — |
| [017](017-sune-temple-service-entry.md) | Sune 神殿服務入口與離開續行 | CONFORMED＋DRAFT | `cmd/pool-game/main.go` | `cmd/pool-disp-scan/main_test.go`、`cmd/pool-game/main_test.go` |
| [018](018-save-hp-status-and-temple-wounds.md) | 存檔生命值／狀態與神殿傷勢治療 | CONFORMED＋DRAFT | — | — |
| [019](019-pool-ecl-dialect-newecl.md) | Pool ECL dialect 與 `20h` 跨 block 交接 | CONFORMED＋DRAFT | 共用 engine | — |
| [020](020-compare-and.md) | `14h COMPARE AND` 四 operand 比較 | CONFORMED | 共用 engine | — |
| [021](021-load-character-selection.md) | `0Ah LOAD CHARACTER` 選定角色與 Pool 記憶體投影 | CONFORMED＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/who.go`、`internal/gamepack/intro.go` | `cmd/pool-game/harbour_walk_test.go` |
| [022](022-dos-cell-entry-order.md) | DOS 自由移動的 ECL 入口順序 | CONFORMED | — | — |
| [023](023-city-hall-proclamations.md) | City Hall 公告與選單目的位址 | CONFORMED＋READY | — | — |
| [024](024-city-hall-commission-proclamations.md) | City Hall commission 公告分派 | CONFORMED | — | `cmd/pool-game/coverage_test.go` |
| [025](025-newecl-transition-lifecycle.md) | `NEWECL` 後的 command-set lifecycle | CONFORMED | `cmd/pool-game/main.go` | `cmd/pool-game/playthrough_test.go` |
| [026](026-city-hall-clerk-entry.md) | City Hall clerk office 正常入口 | CONFORMED | — | — |
| [027](027-save-table-opcode.md) | Pool ECL opcode `35h SAVE TABLE` | CONFORMED | 共用 engine | `cmd/pool-doc-index/main_test.go` |
| [028](028-city-hall-reward-commission-loop.md) | City Hall reward／commission 迴圈 | DRAFT | — | — |
| [029](029-city-hall-fresh-party-commission-exit.md) | 全新隊伍 City Hall 委託列舉與離場 | READY＋DRAFT | — | — |
| [030](030-party-strength-vm-contract.md) | `1Dh PARTYSTRENGTH` VM 契約 | CONFORMED＋DRAFT | `internal/character/dos.go` | `internal/character/export_test.go` |
| [031](031-level-one-party-strength-projection.md) | 一級新角色的 `PARTYSTRENGTH` 投影 | CONFORMED＋DRAFT | — | — |
| [032](032-treasure-vm-contract.md) | `27h TREASURE` 八欄請求與墓園順序 | CONFORMED＋DRAFT | — | — |
| [033](033-item3-block33-record-shape.md) | `ITEM3.DAX/33h` 五筆 63-byte 物品紀錄 | CONFORMED＋DRAFT | `cmd/pool-game/tactical.go`、`internal/gamepack/shop.go` | — |
| [034](034-postcombat-treasure-menu-boundary.md) | 戰後戰利品選單與物品鏈移除邊界 | READY＋DRAFT | — | — |
| [035](035-character-item-receive-and-carry.md) | 角色接收物品、16 格上限與力量負重 | CONFORMED＋DRAFT | `internal/character/carry.go`、`internal/gamepack/weapon_stats.go` | — |
| [036](036-combat-postcombat-treasure-dispatch.md) | `24h COMBAT` 的戰鬥／神殿／戰後服務分派 | CONFORMED＋DRAFT | — | — |
| [037](037-campaign-save-and-ecl-session.md) | 戰役存檔、地圖位置與 ECL session 續點 | CONFORMED＋DRAFT | 共用 engine | — |
| [038](038-graveyard-commission-state-producers.md) | 墓園委託旗標 producer 清冊 | CONFORMED＋DRAFT | — | — |
| [039](039-graveyard-seven-pool-treasure.md) | 墓園七種戰利品累積池 | READY | — | — |
| [040](040-seven-currency-pool-take-share.md) | 七種貨幣的 View／Take／Pool／Share | READY | `cmd/pool-game/appraise.go`、`internal/character/export.go`、`internal/gamepack/movement.go` 等 4 個 | — |
| [041](041-city-hall-completion-notification-table.md) | City Hall 完成通知狀態表 | READY＋DRAFT | `internal/gamepack/cityhall.go` | `cmd/pool-game/coverage_test.go`、`cmd/pool-game/playthrough_test.go`、`internal/gamepack/cityhall_test.go` |
| [042](042-dos-ecl-archive-catalog-and-slums-counter.md) | DOS ECL archive catalog 與 Slums 完成計數 | READY＋DRAFT | — | — |
| [043](043-load-files-and-three-wall-slots.md) | LOAD FILES 與三個 WALLDEF slot | READY | `cmd/pool-game/main.go`、`cmd/pool-world-cell-sweep/main.go`、`internal/gamepack/geometry.go` 等 4 個 | `cmd/pool-game/main_test.go`、`internal/gamepack/walls_test.go` |
| [044](044-ecl-archive-campaign-save.md) | ECL archive campaign 存檔 | CONFORMED＋READY | — | — |
| [045](045-cross-archive-newecl.md) | 跨 archive NEWECL 與 New Phlan→Slums | READY＋DRAFT | — | — |
| [046](046-encounter-roster-and-combat-continuation.md) | 遭遇名冊與 COMBAT 續跑契約 | READY＋DRAFT | `cmd/pool-game/tactical.go` | `cmd/pool-game/main_test.go`、`cmd/pool-game/tactical_test.go` |
| [047](047-shared-first-person-inset-fill.md) | 共用第一人稱內框填滿 | READY | `cmd/pool-game/first_person_inset.go`、`cmd/pool-game/main.go`、`internal/assets/camp_fire.go` 等 4 個 | `cmd/pool-game/main_test.go` |
| [048](048-monster-record-and-precombat-staging.md) | 怪物角色記錄與戰鬥前 staging | CONFORMED＋DRAFT | — | — |
| [049](049-pool-character-combat-fields.md) | Pool 285-byte 角色／怪物戰鬥欄位 | READY＋DRAFT | `internal/gamepack/class_thac0.go`、`internal/gamepack/cloud.go` | `internal/gamepack/cloud_test.go` |
| [050](050-basic-attack-and-damage-roll.md) | Pool 基礎命中與傷害骰 primitive | CONFORMED＋DRAFT | `cmd/pool-game/tactical.go` | — |
| [051](051-attack-slot-phase-count.md) | Pool 雙攻擊槽與半回合攻擊次數 | CONFORMED＋DRAFT | `cmd/pool-game/tactical.go`、`internal/gamepack/monster.go`、`internal/gamepack/saving_throw_table.go` | `cmd/pool-game/attack_forms_test.go`、`cmd/pool-game/main_test.go`、`internal/gamepack/monster_test.go` |
| [052](052-combat-initiative-order.md) | Pool 戰鬥先攻分數與行動者選取 | CONFORMED＋DRAFT | `cmd/pool-game/tactical.go`、`internal/character/export.go`、`internal/gamepack/monster.go` | `internal/gamepack/monster_test.go` |
| [053](053-tactical-movement-budget.md) | Pool 戰術移動預算與八方向成本 | CONFORMED＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/manual_aim.go`、`cmd/pool-game/tactical.go` 等 5 個 | `cmd/pool-game/tactical_test.go` |
| [054](054-softworld-manual-journal-corpus.md) | 《軟體世界》說明書轉錄與 Journal 繁中 corpus | CONFORMED＋DRAFT | — | `internal/journal/journal_test.go` |
| [055](055-ecl-area-script-map.md) | ECL 區域腳本對照 | READY＋DRAFT | `cmd/pool-map-names/main.go`、`internal/gamepack/cityhall.go` | `cmd/pool-map-names/main_test.go`、`internal/gamepack/cityhall_test.go` |
| [056](056-tactical-nearby-occupancy.md) | 戰術鄰近格位查詢、朝向弧與敵對側篩選 | READY | `internal/combat/facing.go`、`internal/combat/nearby.go`、`internal/combat/occupancy.go` | `internal/combat/occupancy_test.go` |
| [057](057-movement-line-trace.md) | 移動直線追蹤與每步成本 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/combat/nearby.go`、`internal/combat/trace.go` | — |
| [058](058-destination-probe-and-tactical-layout.md) | 目的格探測與戰術層資料版面 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/combat/destination.go` | — |
| [059](059-reaction-attack-gate.md) | 離開威脅區的反應攻擊閘門 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/character/export.go`、`internal/combat/nearby.go` 等 5 個 | `internal/gamepack/effect_list_test.go` |
| [060](060-tactical-map-generation.md) | 戰術地圖的生成 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/assets/combat_terrain.go`、`internal/combat/indoormap.go` 等 4 個 | `cmd/pool-game/playthrough_test.go` |
| [061](061-deployment-and-occupancy.md) | 戰鬥部署與佔用格重建 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/combat/deployment.go` | — |
| [062](062-combat-round-loop.md) | 戰鬥回合迴圈與結束條件 | READY＋DRAFT | `cmd/pool-game/tactical.go`、`internal/combat/round.go` | `cmd/pool-game/combat_effects_test.go`、`cmd/pool-game/coverage_test.go`、`cmd/pool-game/playthrough_test.go` 等 4 個 |
| [063](063-character-base-combat-stats.md) | 角色的基礎 AC、THAC0、移動與武器攻擊數值 | READY＋DRAFT | `cmd/pool-game/character_sheet.go`、`cmd/pool-game/dos_export.go`、`cmd/pool-game/tactical.go` 等 12 個 | `cmd/pool-game/dos_export_test.go`、`cmd/pool-game/tactical_test.go`、`internal/gamepack/experience_test.go` 等 5 個 |
| [064](064-in-game-journal.md) | 遊戲內《探險者手冊》 | CONFORMED | `cmd/pool-journal-corpus/main.go` | — |
| [065](065-weapon-driven-combat-stats.md) | 物品型別表與裝備武器決定的戰鬥數值 | READY＋DRAFT | `cmd/pool-game/cast.go`、`cmd/pool-game/equipment.go`、`cmd/pool-game/tactical.go` 等 7 個 | `cmd/pool-game/tactical_test.go`、`internal/gamepack/monster_test.go` |
| [066](066-three-platform-release.md) | 三平台發行包 | CONFORMED＋DRAFT | — | — |
| [067](067-shop-service-and-stock.md) | 商店服務邊界與進貨清單 | CONFORMED＋DRAFT | `cmd/pool-game/shop.go` | `cmd/pool-game/shop_walk_test.go` |
| [068](068-spell-name-table.md) | 法術名稱表 | CONFORMED＋DRAFT | `cmd/pool-game/spells.go`、`internal/gamepack/spell_dispatch.go`、`internal/gamepack/spell_table.go` | — |
| [069](069-character-effect-list.md) | 角色的效果串列（`.spc`） | CONFORMED＋DRAFT | `cmd/pool-game/combat_effects.go`、`cmd/pool-game/tactical.go`、`internal/character/export.go` 等 9 個 | `cmd/pool-game/combat_effects_test.go`、`internal/character/export_test.go`、`internal/gamepack/effect_names_test.go` 等 4 個 |
| [070](070-memorised-spells.md) | 記憶法術陣列與 1-based 法術編號 | CONFORMED＋DRAFT | `cmd/pool-game/cast.go`、`cmd/pool-game/spells.go`、`internal/gamepack/ecl_operands.go` 等 5 個 | `cmd/pool-game/inn_test.go`、`internal/gamepack/memorisation_test.go` |
| [071](071-experience-and-level-caps.md) | 經驗值門檻表與等級上限 | CONFORMED | `cmd/pool-game/main.go`、`internal/gamepack/spell_slots.go` | `internal/journal/manual_tables_test.go` |
| [072](072-spell-slots-and-wisdom-bonus.md) | 可記憶法術數與睿智加成 | CONFORMED＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/spells.go`、`cmd/pool-game/tactical.go` 等 7 個 | `cmd/pool-game/memorise_test.go`、`internal/gamepack/memorisation_test.go`、`internal/gamepack/saving_throw_table_test.go` |
| [073](073-spell-effect-dispatch-table.md) | 法術效果的派發表 | CONFORMED＋DRAFT | `cmd/pool-game/field_cast.go`、`cmd/pool-game/spells.go`、`cmd/pool-spell-dispatch/main.go` 等 5 個 | `internal/gamepack/spell_parameters_test.go` |
| [074](074-spell-parameters-and-messages.md) | 法術參數表與效果訊息 | CONFORMED＋DRAFT | `cmd/pool-game/cast.go`、`cmd/pool-game/spells.go`、`internal/gamepack/game_clock.go` 等 6 個 | `cmd/pool-disp-scan/main_test.go`、`cmd/pool-game/cast_test.go`、`cmd/pool-game/spells_test.go` 等 6 個 |
| [075](075-saving-throws.md) | 豁免判定 | CONFORMED＋DRAFT | `cmd/pool-game/damage.go`、`cmd/pool-game/main.go`、`cmd/pool-game/tactical.go` 等 8 個 | — |
| [076](076-party-facing-convention.md) | 隊伍朝向的座標系 | CONFORMED | `cmd/pool-game/area_map.go`、`cmd/pool-game/main.go`、`cmd/pool-game/party_panel.go` 等 6 個 | `cmd/pool-game/coverage_test.go`、`cmd/pool-game/party_panel_test.go`、`internal/gamepack/facing_test.go` 等 4 個 |
| [077](077-ecl-opcode-dispatch.md) | ECL 指令的派發鏈與運算元個數 | CONFORMED＋DRAFT | — | — |
| [078](078-encounter-menu-opcode.md) | `29h ENCOUNTER MENU` | CONFORMED＋DRAFT | `cmd/pool-game/encounter.go`、`internal/gamepack/ecl_operands.go`、`internal/gamepack/encounter.go` 等 6 個 | `internal/gamepack/encounter_test.go` |
| [079](079-movement-rate.md) | 移動力 | CONFORMED | `cmd/pool-game/checkparty.go`、`cmd/pool-game/encounter.go`、`cmd/pool-game/tactical.go` 等 6 個 | `cmd/pool-game/equipment_test.go` |
| [080](080-armour-class-from-equipment.md) | 裝備算出來的護甲等級 | READY | `cmd/pool-game/party_panel.go`、`internal/gamepack/armour_class.go`、`internal/gamepack/record_recompute.go` | — |
| [081](081-ecl-program-opcode.md) | `38h PROGRAM` | READY＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/program.go`、`internal/gamepack/ecl_operands.go` 等 5 個 | `cmd/pool-game/playthrough_test.go` |
| [082](082-ecl-text-box-opcodes.md) | 文字框的四條 opcode（`11h`、`12h`、`33h`、`3Dh`） | READY | `cmd/pool-game/main.go`、`internal/gamepack/continue_prompt.go`、`internal/gamepack/ecl_operands.go` 等 4 個 | `cmd/pool-game/main_test.go`、`internal/gametext/catalogue_test.go` |
| [083](083-ecl-current-character-opcodes.md) | `39h WHO` 與 `36h ADD NPC`，以及它們共用的「目前角色」槽 | READY＋DRAFT | `internal/gamepack/ecl_operands.go` | — |
| [084](084-ecl-damage-opcode.md) | `2Eh DAMAGE` | READY＋DRAFT | `cmd/pool-game/damage.go`、`internal/gamepack/damage.go`、`internal/gamepack/ending_scene.go` 等 4 個 | `cmd/pool-game/main_test.go`、`internal/assets/ending_test.go` |
| [085](085-ecl-party-query-opcodes.md) | 三條對隊伍發問的 opcode（`32h`、`22h`、`23h`） | READY | `cmd/pool-game/ecl_party_queries.go`、`internal/gamepack/ecl_party_queries.go`、`internal/gamepack/intro.go` | — |
| [086](086-ecl-parlay-opcode.md) | `2Ch PARLAY` | READY | `cmd/pool-game/main.go`、`cmd/pool-game/parlay.go`、`internal/gamepack/ecl_operands.go` 等 4 個 | `cmd/pool-game/main_test.go` |
| [087](087-ecl-input-opcodes.md) | `0Fh INPUT NUMBER` 與 `10h INPUT STRING` | READY | `cmd/pool-game/ecl_input.go`、`cmd/pool-game/main.go`、`internal/gamepack/ecl_operands.go` 等 4 個 | `cmd/pool-game/coverage_test.go`、`cmd/pool-game/main_test.go` |
| [088](088-ecl-rob-opcode.md) | `28h ROB` | READY＋DRAFT | `cmd/pool-game/rob.go`、`internal/gamepack/intro.go`、`internal/gamepack/rob.go` | — |
| [089](089-ecl-protection-opcode.md) | `3Ch PROTECTION` | READY＋DRAFT | `cmd/pool-game/protection.go`、`internal/gamepack/ecl_operands.go`、`internal/gamepack/intro.go` | — |
| [090](090-ecl-who-opcode.md) | `39h WHO` 與「目前角色」 | READY＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/who.go`、`internal/gamepack/intro.go` | `cmd/pool-game/coverage_test.go`、`cmd/pool-game/main_test.go` |
| [091](091-ecl-add-npc-opcode.md) | `36h ADD NPC` | READY＋DRAFT | `cmd/pool-game/addnpc.go`、`cmd/pool-game/tactical.go`、`internal/gamepack/ecl_operands.go` 等 5 個 | `cmd/pool-game/main_test.go` |
| [092](092-ecl-checkparty-opcode.md) | `1Eh CHECKPARTY` | READY | `cmd/pool-game/checkparty.go`、`internal/gamepack/checkparty.go`、`internal/gamepack/intro.go` | — |
| [093](093-ecl-clock-opcode.md) | `34h ECL CLOCK` | READY＋DRAFT | `cmd/pool-game/ecl_clock.go`、`cmd/pool-game/main.go`、`internal/gamepack/ecl_clock.go` 等 5 個 | `cmd/pool-ecl-audit/main_test.go` |
| [094](094-ecl-spell-search-opcode.md) | `3Bh SPELL` | READY＋DRAFT | `cmd/pool-game/spell_search.go`、`internal/gamepack/intro.go`、`internal/gamepack/memorised_spells.go` | — |
| [095](095-thief-skills.md) | 角色記錄的八個賊技能 | READY＋DRAFT | `cmd/pool-game/checkparty.go`、`cmd/pool-game/door.go`、`internal/character/export.go` 等 7 個 | — |
| [096](096-monster-ai-structure.md) | 怪物 AI 的骨架（overlay-09） | READY | `cmd/pool-game/tactical.go`、`internal/gamepack/tactic_offsets.go`、`internal/gamepack/turn_undead_resolve.go` | `internal/gamepack/tactic_offsets_test.go` |
| [097](097-training-level-up-and-energy-drain.md) | 經驗值、訓練所昇級、生命骰與能量吸取 | CONFORMED＋DRAFT | `cmd/pool-game/addnpc.go`、`cmd/pool-game/main.go`、`cmd/pool-game/tactical.go` 等 13 個 | `cmd/pool-game/cast_test.go`、`cmd/pool-game/door_test.go`、`cmd/pool-game/experience_test.go` 等 6 個 |
| [098](098-spell-casting-machinery.md) | 施法的共用機制（擲骰、施法者等級、處理常式的呼叫慣例） | CONFORMED＋DRAFT | `cmd/pool-game/cast.go`、`cmd/pool-game/main.go`、`cmd/pool-game/spells.go` 等 9 個 | `cmd/pool-game/cast_test.go`、`cmd/pool-game/memorise_test.go`、`internal/gamepack/spell_cast_test.go` |
| [099](099-boat-travel-and-quest-gate.md) | 搭船旅行與它的進度閘門 | CONFORMED | `cmd/pool-game/main.go` | `cmd/pool-game/coverage_test.go` |
| [100](100-the-6dd5-map-exit-gate.md) | `DS:6DD5h` — 擋住整個世界的那一個變數 | CONFORMED | `cmd/pool-disp-scan/main.go`、`cmd/pool-game/main.go`、`cmd/pool-game/training_gate.go` | `cmd/pool-doc-index/main_test.go`、`cmd/pool-game/coverage_test.go`、`cmd/pool-game/playthrough_test.go` |
| [101](101-world-transition-graph.md) | 世界怎麼接起來——NEWECL 圖與邊界出口 | READY | — | `cmd/pool-game/stojanow_gate_test.go`、`cmd/pool-game/wilderness_explore_test.go`、`cmd/pool-world-graph/main_test.go` |
| [102](102-city-location-dispatch.md) | 城區的地點是 terrain 索引分派的，而且有些要面對它 | READY | `cmd/pool-game/main.go`、`cmd/pool-map-names/main.go` | `cmd/pool-game/combat_effects_test.go`、`cmd/pool-game/coverage_test.go`、`cmd/pool-game/harbour_walk_test.go` 等 5 個 |
| [103](103-world-cell-sweep.md) | 整包格子入口掃描 | READY | — | `cmd/pool-game/coverage_test.go` |
| [104](104-script-call-selectors.md) | `2Dh CALL` 的選擇子只有五個有動作 | READY | — | — |
| [105](105-wilderness-map.md) | 野外地圖——座標、三張圖怎麼接、地點表 | READY＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-wilderness-map/main.go`、`internal/gamepack/wilderness.go` | `cmd/pool-game/coverage_test.go`、`cmd/pool-game/harbour_walk_test.go`、`cmd/pool-game/wilderness_cave_test.go` 等 4 個 |
| [106](106-ecl-address-space.md) | ECL 的位址空間是虛擬的 | READY＋DRAFT | `cmd/pool-disp-scan/main.go`、`cmd/pool-game/training_gate.go` | `cmd/pool-disp-scan/main_test.go`、`cmd/pool-game/playthrough_test.go`、`cmd/pool-game/stojanow_gate_test.go` |
| [107](107-newecl-ff-sentinel.md) | `NEWECL FFh` 是「不換區塊」 | READY | 共用 engine | — |
| [108](108-ending-cutscene.md) | 結局過場（overlay-18 entry 1） | READY＋DRAFT | `cmd/pool-game/ending.go`、`cmd/pool-game/main.go`、`cmd/pool-game/program.go` 等 6 個 | `cmd/pool-game/playthrough_test.go`、`internal/assets/ending_test.go`、`internal/gametext/catalogue_test.go` |
| [109](109-overlay-stub-segments.md) | overlay 的 stub segment 對照表 | READY | `cmd/pool-game/main.go`、`cmd/pool-game/party_menu.go`、`internal/gamepack/overlay_segments.go` | `internal/gamepack/overlay_segments_test.go` |
| [110](110-spellbook.md) | 法術書（角色記錄 `+32h + 編號`） | CONFORMED＋DRAFT | `cmd/pool-game/spellbook.go`、`cmd/pool-game/spells.go`、`internal/gamepack/spellbook.go` 等 4 個 | `cmd/pool-game/memorise_test.go`、`cmd/pool-game/spellbook_test.go`、`internal/gamepack/spellbook_test.go` |
| [111](111-turn-undead.md) | 轉變不死生物 | READY | `internal/gamepack/turn_undead_resolve.go` | `internal/journal/manual_tables_test.go` |
| [112](112-effect-code-dispatch.md) | 效果代碼的分派（overlay-24 entry 3／entry 1 與 `014Dh`） | READY | `cmd/pool-game/tactical.go`、`internal/gamepack/cloud.go`、`internal/gamepack/effect_list.go` 等 4 個 | `cmd/pool-game/cast_test.go`、`internal/gamepack/effect_list_test.go` |
| [113](113-game-pack.md) | Pool 的 game pack 與 adapter | READY | — | — |
| [114](114-camp-rest-time.md) | 遊戲時鐘與紮營的休息時間（overlay-20） | READY | `cmd/pool-game/camp.go`、`cmd/pool-game/main.go`、`cmd/pool-game/text.go` 等 5 個 | `cmd/pool-disp-scan/main_test.go`、`cmd/pool-game/camp_interruption_test.go`、`cmd/pool-game/camp_screen_test.go` 等 5 個 |
| [115](115-temple-services.md) | 神殿的九項服務（overlay-04） | READY | `cmd/pool-game/main.go`、`internal/save/state.go` | `cmd/pool-game/main_test.go` |
| [116](116-appraise-and-sell.md) | 估價與販賣寶石珠寶（overlay-21 entry 19） | READY＋OPEN | `cmd/pool-game/appraise.go`、`cmd/pool-game/shop.go`、`cmd/pool-game/text.go` | `cmd/pool-game/camp_test.go` |
| [117](117-npc-approach-portrait.md) | APPROACH 的 NPC 半身像 | READY＋DRAFT | `cmd/pool-game/main.go`、`internal/assets/camp_fire.go`、`internal/assets/npc_portrait.go` 等 5 個 | `cmd/pool-disp-scan/main_test.go`、`internal/gamepack/intro_test.go` |
| [118](118-adventure-status-line-and-clock.md) | 冒險畫面的狀態列與遊戲時鐘 | READY＋DRAFT | `cmd/pool-game/main.go`、`cmd/pool-game/party_panel.go`、`internal/save/state.go` | `cmd/pool-game/combat_effects_test.go`、`cmd/pool-game/party_panel_test.go`、`internal/gamepack/effect_time_test.go` |
| [119](119-adventure-command-bar.md) | 冒險畫面的指令列與平面全圖 | READY＋DRAFT | `cmd/pool-game/area_map.go`、`cmd/pool-game/camp.go`、`cmd/pool-game/command_bar.go` 等 8 個 | `cmd/pool-game/area_map_test.go`、`cmd/pool-game/field_cast_test.go` |
| [120](120-wall-symbol-bands.md) | 8×8 符號的五帶 | READY＋DRAFT | `cmd/pool-game/area_map.go`、`cmd/pool-game/main.go`、`cmd/pool-game/screen_frame.go` 等 5 個 | `cmd/pool-game/first_person_inset_test.go` |
| [121](121-cloud-objects.md) | 盤面上的雲團物件 | READY＋DRAFT | `cmd/pool-game/cast.go`、`cmd/pool-game/cloud.go`、`cmd/pool-game/tactical.go` 等 6 個 | `cmd/pool-game/cloud_test.go`、`internal/gamepack/cloud_test.go`、`internal/gamepack/spell_cast_test.go` |
| [122](122-locked-doors.md) | 鎖住的門（`Bash`／`Pick`／`Knock`） | CONFORMED＋DRAFT | `cmd/pool-disp-scan/main.go`、`cmd/pool-game/door.go`、`cmd/pool-game/main.go` 等 4 個 | `cmd/pool-doc-index/main_test.go`、`internal/gamepack/door_test.go` |
| [123](123-screen-frame.md) | 畫面外框的繩索花紋 | CONFORMED＋DRAFT | `cmd/pool-game/command_bar.go`、`cmd/pool-game/main.go`、`cmd/pool-game/screen_frame.go` | `cmd/pool-game/screen_frame_test.go`、`cmd/pool-game/tactical_test.go` |
| [124](124-map-names-from-the-original.md) | 地圖的名字要從原版自己的文字取 | READY＋DRAFT | — | — |
| [125](125-map-boundary-and-the-exit-flag.md) | 走到地圖邊界會怎樣——`@6DD5` 是誰寫的 | CONFORMED | `cmd/pool-disp-scan/main.go`、`cmd/pool-game/main.go` | `cmd/pool-disp-scan/main_test.go` |
| [126](126-first-person-inset-pixel-parity.md) | 第一人稱內框逐格對上原版 | CONFORMED＋DRAFT | `cmd/pool-game/first_person_inset.go` | `cmd/pool-game/first_person_inset_test.go` |
| [127](127-aim-bar-and-manual-cursor.md) | 瞄準列與 Manual 格子游標（overlay-13 `352Ch`） | CONFORMED | `cmd/pool-game/cast.go`、`cmd/pool-game/main.go`、`cmd/pool-game/manual_aim.go` 等 4 個 | `cmd/pool-game/manual_aim_test.go`、`cmd/pool-game/music_test.go` |
| [128](128-music-cues.md) | 配樂——素材、派曲的形狀，以及界線在哪 | CONFORMED | `cmd/pool-game/main.go`、`cmd/pool-game/music.go`、`internal/music/catalog.go` | `internal/music/catalog_test.go`、`internal/music/render_test.go` |
| [129](129-combat-screen-layout-and-command-bar.md) | 戰鬥畫面的版面與指令列 | READY＋DRAFT | `cmd/pool-game/combat_screen.go`、`cmd/pool-game/main.go`、`cmd/pool-game/screen_frame.go` 等 7 個 | `cmd/pool-game/combat_screen_test.go`、`cmd/pool-game/tactical_test.go`、`internal/assets/monster_sprite_test.go` 等 4 個 |
| [130](130-character-sheet-layout.md) | 人物資料頁的版面 | READY＋DRAFT | `cmd/pool-game/character_sheet.go`、`cmd/pool-game/command_bar.go`、`cmd/pool-game/main.go` 等 5 個 | `cmd/pool-game/field_cast_test.go` |
| [131](131-combat-terrain-tiles.md) | 戰場的地形圖塊 | READY＋DRAFT | `cmd/pool-game/combat_screen.go`、`cmd/pool-game/sprite_overview.go`、`internal/assets/combat_terrain.go` | — |
| [132](132-journal-citations-open-in-place.md) | 文字報了手冊編號就直接翻過去 | READY | `cmd/pool-game/journal_link.go`、`cmd/pool-game/main.go` | — |
| [133](133-original-keyboard-shape.md) | 原版怎麼讀鍵盤 | READY＋DRAFT | — | — |
| [134](134-spell-list-layout.md) | 原版的法術清單版面 | READY＋DRAFT | `cmd/pool-game/field_cast.go`、`cmd/pool-game/screen_state.go` | `cmd/pool-game/spell_page_test.go` |
| [135](135-camp-screen-layout.md) | 原版紮營畫面的版面 | READY | `cmd/pool-game/camp.go`、`cmd/pool-game/command_bar.go`、`cmd/pool-game/icon_menu.go` 等 7 個 | `cmd/pool-game/camp_screen_test.go`、`cmd/pool-game/inn_test.go`、`cmd/pool-game/memorise_test.go` 等 4 個 |

## `cmd/` 底下的工具

| 工具 | 做什麼 | 測試 | 相關規格 |
|---|---|---|---|
| `dos-screen-text` | reads an unscaled 320x200 DOS text grid captured at an integer nearest-neighbour scale | 有 | — |
| `export-title` | 把原版的標題畫面從 DAX 解出來寫成 PNG，給對拍與說明文件用 | — | 001 |
| `pool-city-hall-audit` | derives a reproducible structural inventory of the City Hall reward and commission loops from the original block-8 trace | 有 | — |
| `pool-combat-icon-audit` | inventories Pool's complete combat-icon archives through the reusable engine decoder | — | — |
| `pool-disp-scan` | 找 overlay 與 START.EXE 裡對某個位址／位移的記憶體存取 | 有 | 017、074、100、106、114、117 等 8 份 |
| `pool-doc-index` | 產生 docs/spec/000-index.md：每份規格的狀態、實作它的檔案、釘住它的測試，以及 cmd/ 底下每一支工具在做什麼 | 有 | 027、100、122 |
| `pool-ecl-audit` | measures the reusable ECL decoder against every Pool ECL block | 有 | 002、093 |
| `pool-ecl-frontier` | lists the ECL opcodes that appear in Pool blocks but have neither a core VM handler nor an adapter passthrough, with every call site | 有 | 002 |
| `pool-ecl-memory-audit` | inventories raw ECL operand references to selected runtime addresses across every Pool ECL archive | 有 | — |
| `pool-ecl-opcodes` | 把 overlay-03 的 ECL 派發鏈 dump 成 JSON：每條 opcode 的處理常式位移與運算元個數，並標出與共用 engine 那張二手 arity 表的差異 | — | — |
| `pool-ecl-trace` | exports one original Pool ECL block's complete statically reachable graph without executing or assigning story semantics | 有 | — |
| `pool-font-coverage` | 報出遊戲要顯示、但倚天字型畫不出來的字 | — | — |
| `pool-game` | remake 的遊戲本體：Ebiten 視窗、玩家輸入、畫面，以及與共用 engine 和 game pack 的接線 | 有 | 003、005、007、008、015、016 等 89 份 |
| `pool-geo-audit` | decodes every Pool GEO block through the shared engine and records only structural map evidence | — | — |
| `pool-initial-cell-sweep` | executes the original initial-map cell lifecycle entry against isolated copies of the post-Rolf VM state | 有 | — |
| `pool-input-manifest` | inventories the fixed DOS source ZIP without extracting or modifying its contents | 有 | — |
| `pool-inventory` | performs a read-only shape audit of a DOS game ZIP | — | — |
| `pool-journal-corpus` | 把轉錄好的《探險者手冊》上冊切成遊戲內可查的條目 | — | 064 |
| `pool-map-names` | 把「換圖的目的地區塊」與「同一段腳本剛印出來的字」配成對，用來替每一張地圖找出**原版自己給的名字** | 有 | 055、102 |
| `pool-name-audit` | 把說明書定案的專有名詞回對原版資料自己的字串 | 有 | — |
| `pool-ovr-manifest` | 產生 docs/audit/dos-ovr-manifest.json：38 顆 overlay 的位置、長度、重定位表與各自的 SHA-256，是所有 overlay 反查的起點 | 有 | — |
| `pool-portrait-audit` | measures Pool's HEAD/BODY archives through the reusable engine picture decoder | 有 | — |
| `pool-spell-dispatch` | 把 overlay-22 的法術效果派發表 dump 成 JSON，供 spec 073 引用，也當作後續逐支解讀處理常式的工作清單 | — | 073 |
| `pool-text-inventory` | 盤點原版 ECL 裡所有玩家看得到的敘述文字 | — | 002 |
| `pool-wilderness-map` | 解出野外地圖上「哪一格有東西」的表 | 有 | 105 |
| `pool-worklist` | 管未完成項 | 有 | — |
| `pool-world-cell-sweep` | 把每一張地圖的每一格都餵進它自己的 ECL 區塊，跑入口 0 再跑入口 1，記下停在哪一種邊界 | 有 | 043 |
| `pool-world-graph` | 把「世界怎麼接起來」從原始資料量出來：每一個 ECL 區塊會 NEWECL 到哪些區塊、載入哪些檔案，以及每一張 GEO 地圖的邊界上哪些格子往外沒有牆 | 有 | 002、076、101 |
