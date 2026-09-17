"""原版（dosgolem）的互動駕駛：一個常駐的 `shots -serve`，看記憶體與畫面再決定下一個鍵。

在容器裡跑（tools/dosgolem-cheat-playthrough.py 會自己起容器）：
  /bin/shots   dosgolem 的 shots（靜態編譯）
  /orig        原版目錄（唯讀）
  /work        workplace/dosgolem-cheat（狀態檔、暫存層、字形表、GEO 匯出）

位址（spec 106／015／107）：
  DS:6A0B  x, y, 朝向(0..7), ?, 地形      DS:52D4  ECL archive      DS:82A2  ECL block
  DS:4954  5 = 戰鬥中                      DS:4933／4937  class 0／1 的 far pointer
  class 0 位址 a → [4933h] + (a-4900h)*2；class 1 位址 a → [4937h] + (a-6B00h)*2"""
import json, os, subprocess, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import dos_text

DS = 0x0850
DAMAGE_HOOK = 'stub=010A:00AC,hp=11B,side=10E,combat=4954:5,party=0,lock,kill'


class Dos:
    def __init__(self, work='/work', load=None, keys='', log=None, intercept=True):
        self.work = work
        self.font = dos_text.load_font(os.path.join(work, 'font.json'))
        self.log = log or sys.stdout
        # 狀態檔記著暫存層的絕對路徑（開著的 game.ovr），所以固定掛在 /scratch。
        scratch = '/scratch'
        args = ['/bin/shots', '-exe', '/orig/start.exe', '-root', '/orig', '-scratch', scratch, '-serve',
                '-idle', '3000000', '-budget', '400000000']
        if intercept:
            args += ['-intercept-damage', DAMAGE_HOOK]
        if load:
            args += ['-load-state', load]
        if keys:
            args += ['-keys', keys]
        # stderr 併進 stdout：shots 若 panic，訊息才會進段落日誌。
        self.proc = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                     text=True, bufsize=1)
        self.keys = []          # 這一次送過的鍵（收據用）
        self.damage = []        # 攔截紀錄
        self.steps = 0
        self._reply(None)

    def _reply(self, command):
        if command is not None:
            if os.environ.get('DOS_TRACE'):
                self.log.write('  > %s\n' % command)
            self.proc.stdin.write(command + '\n')
            self.proc.stdin.flush()
        while True:
            line = self.proc.stdout.readline()
            if not line:
                raise RuntimeError('shots 結束了（命令 %r）' % command)
            if not line.startswith('@@ '):
                if not line.startswith(('  →', '    ', '送 ')):
                    self.log.write('  | ' + line)
                continue
            if line.startswith('@@ '):
                reply = json.loads(line[3:])
                if reply.get('error'):
                    raise RuntimeError('shots：%s（命令 %r）' % (reply['error'], command))
                self.damage += reply.get('damage') or []
                self.steps = reply.get('steps', self.steps)
                return reply

    def key(self, key):
        self.keys.append(key)
        return self._reply('key ' + key)

    def wait(self):
        return self._reply('wait')

    def poke(self, seg, off, data):
        return self._reply('poke %04X:%04X %s' % (seg, off, bytes(data).hex()))

    def mem(self, seg, off, n):
        name = '%04X:%X' % (seg, off)
        value = self._reply('peek %s:%d' % (name, n))['peek']
        return bytes.fromhex(list(value.values())[0].split('|')[1])

    def word(self, seg, off):
        b = self.mem(seg, off, 2)
        return b[0] | b[1] << 8

    def far(self, off):
        b = self.mem(DS, off, 4)
        return (b[2] | b[3] << 8), (b[0] | b[1] << 8)

    def linear_class0(self, addr):
        seg, off = self.far(0x4933)
        return seg * 16 + ((off + (0x6E00 + addr * 2)) & 0xFFFF)

    def class0(self, addr):
        seg, off = self.far(0x4933)
        return self.mem(seg, (off + 0x6E00 + addr * 2) & 0xFFFF, 1)[0]

    def class0_many(self, addrs):
        seg, off = self.far(0x4933)
        return {a: self.mem(seg, (off + 0x6E00 + a * 2) & 0xFFFF, 1)[0] for a in addrs}

    def class1(self, addr):
        seg, off = self.far(0x4937)
        return self.mem(seg, (off + 0x2A00 + addr * 2) & 0xFFFF, 1)[0]

    def where(self):
        p = self.mem(DS, 0x6A0B, 5)
        return {'archive': self.mem(DS, 0x52D4, 1)[0], 'block': self.mem(DS, 0x82A2, 1)[0],
                'x': p[0], 'y': p[1], 'facing': p[2] // 2, 'terrain': p[4],
                'combat': self.mem(DS, 0x4954, 1)[0] == 5,
                # LOAD FILES 把 GEO 區塊編號寫在隊伍記錄 +18Ah（spec 043），也就是 class 0 的 49C5h。
                'geo': self.class0(0x49C5)}

    def frame(self):
        path = os.path.join(self.work, 'now.idx')
        self._reply('frame ' + path)
        return open(path, 'rb').read()

    def save(self, path):
        self._reply('save ' + path)

    def quit(self):
        try:
            self._reply('quit')
        except RuntimeError:
            pass
        self.proc.wait(timeout=30)


def fg_color(idx, column, row):
    values = [idx[(row * 8 + y) * 320 + column * 8 + x] & 0x0F for y in range(8) for x in range(8)]
    counts = {}
    for v in values:
        counts[v] = counts.get(v, 0) + 1
    background = max(counts, key=lambda v: (counts[v], -values.index(v)))
    rest = {v: c for v, c in counts.items() if v != background}
    return max(rest, key=rest.get) if rest else None


class Screen:
    """一幀的文字：rows[0..24]，底列的選項與熱鍵（白色字母）。"""

    def __init__(self, idx, font):
        self.idx = idx
        self.rows = dos_text.read_rows(idx, font)

    def text(self, first=0, last=24):
        return '\n'.join(r for r in self.rows[first:last + 1])

    def flat(self, first=16, last=23):
        """文字框接成一行：去掉左右邊框那一欄、行與行之間補一個空格，跨行的句子才比得到。"""
        return ' '.join(' '.join(r[1:39].split()) for r in self.rows[first:last + 1] if r[1:39].strip())

    def has(self, needle):
        return any(needle in r for r in self.rows)

    def bottom(self):
        return self.rows[24]

    def options(self, row=24):
        """底列的選項：[(文字, 熱鍵字母)]。熱鍵是那一個詞裡顏色跟其他字不同的字母（白色 15）。"""
        line = self.rows[row]
        out = []
        column = 0
        while column < len(line):
            if line[column] == ' ':
                column += 1
                continue
            end = column
            while end < len(line) and line[end] != ' ':
                end += 1
            word = line[column:end]
            colors = [fg_color(self.idx, c, row) for c in range(column, end)]
            hot = None
            for i, color in enumerate(colors):
                if color == 15 and any(other != 15 for other in colors):
                    hot = word[i]
                    break
            out.append((word, hot, column))
            column = end
        return out
