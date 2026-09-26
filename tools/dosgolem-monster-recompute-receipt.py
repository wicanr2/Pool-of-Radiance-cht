#!/usr/bin/env python3
"""把獸人家開打那一幀的 combatant 記錄（dosgolem 讀出的 120h bytes，spec 063 的
docs/audit/dosgolem-monster-thac0-runtime.json 同一批）裡 overlay-25 entry 7 寫的欄位
抄成收據：docs/audit/dosgolem-monster-recompute-runtime.json（#93，spec 147）。

輸入是 tools/dosgolem-drive-orc-home.py 那一套留下的 workplace/orc-drive/thac0-recs/shots.json
（從 orc1.state 接著跑、-peek 每一筆記錄 120h bytes）。只收怪物（`+10Eh` 讀不到，
名字不是單一字母的那幾筆）；隊員的建角值不在這張收據的範圍。

用法：python3 tools/dosgolem-monster-recompute-receipt.py <shots.json> <輸出.json>"""
import hashlib, json, sys

src, dst = sys.argv[1], sys.argv[2]
raw = open(src, 'rb').read()
shots = json.loads(raw)
shot = shots[-1]
records = []
index = 0
for key, value in shot['peek'].items():
    if key.startswith('ds:'):
        continue
    index += 1  # 讀取順序就是 DS:6517h 指標表的順序，也就是 combatant 索引（1 起）
    segment, hexbytes = value.split('|')
    b = bytes.fromhex(hexbytes)
    assert len(b) == 0x120, (key, len(b))
    name = b[1:1 + b[0]].decode('latin1')
    if len(name) == 1:  # 隊員 B..F
        continue
    records.append({
        'index': index,
        'record': key,
        'name': name,
        'base_2Dh': b[0x2D],
        'base_ac_A9h': b[0xA9],
        'base_move_72h': b[0x72],
        'ability_flag_AAh': b[0xAA],
        'dice_A2h_A8h': b[0xA2:0xA9].hex(),
        'thac0_110h': b[0x110],
        'ac_111h': b[0x111],
        'rear_ac_112h': b[0x112],
        'runtime_dice_114h_11Ah': b[0x114:0x11B].hex(),
        'move_11Ch': b[0x11C],
        'carried_102h': b[0x102] | b[0x103] << 8,
        'weapon_0CCh': b[0xCC:0xD0].hex(),
        'slot_0F8h': b[0xF8:0xFC].hex(),
        'slot_0FCh': b[0xFC:0x100].hex(),
        'item_count_C7h': b[0xC7],
    })
out = {
    'schema': 'pool-dosgolem-monster-recompute/1',
    'generator': 'dosgolem',
    'generator_revision': '9bd269a',
    'original_exe_sha256': '12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f',
    'source_shots_sha256': hashlib.sha256(raw).hexdigest(),
    'state': '獸人的家 (3,3) N 開打那一幀（orc1.state，與 dosgolem-monster-thac0-runtime.json 同一批讀取）',
    'frame': {'label': shot['label'], 'step': shot['step'], 'screen_sha256': shot['sha256']},
    'how': '每一筆 combatant 記錄 -peek <seg>:<off>:120h；這裡只抄怪物、只抄 overlay-25 entry 7（0BBEh）寫的欄位與它讀的基礎值',
    'records': records,
}
json.dump(out, open(dst, 'w'), ensure_ascii=False, indent=1)
print('%d 筆怪物記錄 → %s' % (len(records), dst))
