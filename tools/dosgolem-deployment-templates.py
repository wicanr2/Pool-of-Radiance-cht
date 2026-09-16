#!/usr/bin/env python3
"""讀原版開打那一幀填好的陣型樣板（DS:43A2h，2 組 × 4 陣型 × 6 列 × 11 欄 = 528 bytes）
與它的來源表 DS:304h（五組列範圍，60 bytes 當正對照），寫成
docs/audit/dosgolem-deployment-templates.json（#35，spec 061）。

三場貧民窟固定事件用 tools/dosgolem-drive-orc-home.py 留在 workplace/orc-drive 的戰鬥畫面
狀態檔（orc1／guards2／alarm1），哥布林那一場從開機重走收據裡的鍵序到開打。"""
import json, os, sys, importlib.util
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
spec = importlib.util.spec_from_file_location('drv', os.path.join(ROOT, 'tools', 'dosgolem-drive-orc-home.py'))
drv = importlib.util.module_from_spec(spec); spec.loader.exec_module(drv)
drv.PEEK = "ds:2D0:16,ds:304:60,ds:6A0B:3,ds:45B2:9,ds:6772:2,ds:43A2:528"
# 正對照：ida-start-deployment-tables.json 是 DS:2D0h 起 112 bytes，304h 從第 52 個 byte 起。
STATIC = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'ida-start-deployment-tables.json')))
assert STATIC['ds_offset'] == 0x2D0 and STATIC['length'] == 112
EXPECT_304 = STATIC['bytes'][52 * 2:]

def frame_templates(shot):
    p = shot['peek']
    seg, spans = p['ds:304'].split('|')
    seg2, tpl = p['ds:43A2'].split('|')
    assert seg == seg2 == '0850', 'DS 抓錯：%s %s' % (seg, seg2)
    assert spans == EXPECT_304, 'DS:304h 執行期與 START.EXE 不同：%s vs %s' % (spans, EXPECT_304)
    return {'ds_304': p['ds:304'], 'ds_43A2': p['ds:43A2'], 'party_cell': p['ds:6A0B'], 'offsets_45B2': p['ds:45B2'], 'counts_6772': p['ds:6772']}

out = {'schema': 'pool-dosgolem-templates/1', 'generator': 'dosgolem',
       'original_exe_sha256': '12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f',
       'peek_spec': drv.PEEK, 'fights': {}}
goblins = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-peek-goblins.json')))
for name, state in (('orc-home', 'orc1.state'), ('guards', 'guards2.state'), ('alarm', 'alarm1.state')):
    shots = drv.run('d', state, None, 'tpl-' + name)
    out['fights'][name] = dict(frame_templates(shots[0]), state=state, frame=shots[0]['label'])
    print(name, out['fights'][name]['party_cell'], out['fights'][name]['offsets_45B2'])
keys = goblins['keys'].split(',')
keys = keys[:keys.index('c', keys.index('Left')) + 3]  # 到 c,Return,Return 開打
shots = drv.run(','.join(keys), None, None, 'tpl-goblins')
out['fights']['goblins'] = dict(frame_templates(shots[-1]), keys=','.join(keys), frame=shots[-1]['label'])
print('goblins', out['fights']['goblins']['party_cell'], out['fights']['goblins']['offsets_45B2'])
rev = os.popen('git -C %s rev-parse --short HEAD' % drv.DOSGOLEM).read().strip()
out['generator_revision'] = rev
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-templates.json'), 'w'), ensure_ascii=False, indent=1)
print('寫出 docs/audit/dosgolem-deployment-templates.json')
