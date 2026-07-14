// AdminApp — trình sửa story (React Flow). Gate bằng admin token; load StoryFile qua
// GET /admin/models/{id}/content, sửa đồ thị, lưu qua PUT .../content (seed.Replace validate).
// Lưu ý: Replace tạo model id MỚI → sau khi lưu ta reload danh sách model theo `code`.
import { useCallback, useEffect, useMemo, useState } from 'react';

import Box from '@mui/material/Box';
import Button from '@mui/material/Button';

import { admin, adminToken } from '@models/admin';
import FlowEditor from './components/flowEditor';
import { buildPlayerMap } from './layout';

const SCENE_TYPES = ['linear', 'choice', 'ending'];

// ---- input dùng chung ----
const fieldSx = {
  width: '100%', bgcolor: '#0b0e14', color: '#e9eef8', border: '1px solid #2a3547',
  borderRadius: 1, px: 1, py: 0.75, fontSize: 13, fontFamily: 'inherit', outline: 'none',
  '&:focus': { borderColor: '#fe2d9b' },
};
const Label = ({ children }) => (
  <Box sx={{ fontSize: 11, fontWeight: 700, letterSpacing: '.5px', textTransform: 'uppercase', color: '#8aa0c0', mt: 1.5, mb: 0.5 }}>{children}</Box>
);

// JSON field: parse khi hợp lệ mới apply; báo đỏ nếu sai.
function JsonField({ value, onApply, rows = 3 }) {
  const [txt, setTxt] = useState('');
  const [bad, setBad] = useState(false);
  useEffect(() => { setTxt(value == null ? '' : JSON.stringify(value, null, 1)); setBad(false); }, [value]);
  return (
    <Box
      component="textarea"
      rows={rows}
      value={txt}
      onChange={(e) => {
        const v = e.target.value; setTxt(v);
        if (v.trim() === '') { setBad(false); onApply(undefined); return; }
        try { const o = JSON.parse(v); setBad(false); onApply(o); } catch { setBad(true); }
      }}
      sx={{ ...fieldSx, resize: 'vertical', fontFamily: 'ui-monospace,monospace', fontSize: 11.5, borderColor: bad ? '#ff5470' : '#2a3547' }}
    />
  );
}

export default function AdminApp() {
  const [token, setToken] = useState(adminToken.get());
  const [models, setModels] = useState(null); // null = chưa load, [] = trống
  const [modelId, setModelId] = useState(null);
  const [story, setStory] = useState(null);
  const [chIdx, setChIdx] = useState(0);
  const [sel, setSel] = useState(null);
  const [status, setStatus] = useState('');
  const [err, setErr] = useState('');
  const [dirty, setDirty] = useState(false);

  const loadModels = useCallback(async (preferCode) => {
    setErr('');
    try {
      const { models: ms } = await admin.models();
      setModels(ms);
      const pick = (preferCode && ms.find((m) => m.code === preferCode)) || ms[0];
      if (pick) { setModelId(pick.id); return pick.id; }
    } catch (e) { setErr(e.message || 'Không tải được models — kiểm tra admin token.'); setModels([]); }
    return null;
  }, []);

  const loadContent = useCallback(async (id) => {
    setErr(''); setStatus('Đang tải…');
    try {
      const sf = await admin.content(id);
      setStory(sf); setDirty(false); setSel(null); setStatus('');
    } catch (e) { setErr(e.message || 'Không tải được nội dung.'); setStatus(''); }
  }, []);

  useEffect(() => { if (token) loadModels(); }, []); // eslint-disable-line
  useEffect(() => { if (modelId != null) loadContent(modelId); }, [modelId, loadContent]);

  const chapter = story?.chapters?.[chIdx];
  const sceneCodes = useMemo(
    () => (story?.chapters || []).flatMap((c) => (c.scenes || []).map((s) => s.code)),
    [story],
  );

  const updateChapter = useCallback((newCh) => {
    setStory((s) => {
      const chapters = s.chapters.slice();
      chapters[chIdx] = newCh;
      return { ...s, chapters };
    });
    setDirty(true);
  }, [chIdx]);

  // ---- mutators trên scene/choice của chapter hiện tại ----
  const patchScene = (code, patch) => {
    updateChapter({ ...chapter, scenes: chapter.scenes.map((s) => (s.code === code ? { ...s, ...patch } : s)) });
  };
  const patchChoice = (code, ci, patch) => {
    updateChapter({
      ...chapter,
      scenes: chapter.scenes.map((s) => (s.code === code
        ? { ...s, choices: s.choices.map((c, i) => (i === ci ? { ...c, ...patch } : c)) } : s)),
    });
  };
  const addScene = () => {
    const base = 'scene_'; let n = 1;
    while (sceneCodes.includes(base + n)) n += 1;
    const code = base + n;
    updateChapter({ ...chapter, scenes: [...chapter.scenes, { code, type: 'linear', next: '' }] });
    setSel({ kind: 'scene', code });
  };
  const deleteSel = () => {
    if (!sel) return;
    if (sel.kind === 'scene') {
      updateChapter({ ...chapter, scenes: chapter.scenes.filter((s) => s.code !== sel.code) });
    } else if (sel.kind === 'choice') {
      updateChapter({ ...chapter, scenes: chapter.scenes.map((s) => (s.code === sel.code
        ? { ...s, choices: s.choices.filter((_, i) => i !== sel.choiceIdx) } : s)) });
    }
    setSel(null);
  };

  // Sinh bản đồ người-chơi từ story graph (dùng vị trí đã kéo) → lưu map-only (không churn id).
  const genPlayerMap = async () => {
    if (!chapter) return;
    setErr(''); setStatus('Đang sinh bản đồ…');
    try {
      const map = buildPlayerMap(chapter);
      const { chapters: chs } = await admin.chapters(modelId);
      const dbCh = chs.find((c) => c.idx === chapter.idx);
      if (!dbCh) throw new Error('không tìm thấy chapter trong DB');
      await admin.saveChapterMap(dbCh.id, map);
      // đồng bộ in-memory (không đánh dấu dirty vì đã lưu qua map endpoint).
      setStory((s) => {
        const chapters = s.chapters.slice();
        chapters[chIdx] = { ...chapters[chIdx], map };
        return { ...s, chapters };
      });
      setStatus('Đã sinh bản đồ ✓'); setTimeout(() => setStatus(''), 2500);
    } catch (e) { setStatus(''); setErr(e.message || 'Sinh bản đồ thất bại'); }
  };

  const save = async () => {
    setErr(''); setStatus('Đang lưu…');
    try {
      await admin.saveContent(modelId, story);
      const code = models.find((m) => m.id === modelId)?.code;
      const newId = await loadModels(code); // id đổi sau Replace
      if (newId != null) await loadContent(newId);
      setStatus('Đã lưu ✓'); setTimeout(() => setStatus(''), 2500);
    } catch (e) {
      setStatus('');
      setErr(e.code === 'CONTENT_INVALID' ? `Nội dung không hợp lệ: ${e.message}` : (e.message || 'Lưu thất bại'));
    }
  };

  // ---- token gate ----
  if (!token || (models && models.length === 0 && err)) {
    return (
      <Box sx={{ height: '100%', display: 'grid', placeItems: 'center', bgcolor: '#0b0e14', color: '#e9eef8', fontFamily: 'ui-sans-serif,system-ui,sans-serif' }}>
        <Box sx={{ width: 340, p: 3, bgcolor: '#11161f', border: '1px solid #2a3547', borderRadius: 2 }}>
          <Box sx={{ fontSize: 16, fontWeight: 700, mb: 0.5 }}>Story Editor · Admin</Box>
          <Box sx={{ fontSize: 12.5, color: '#8aa0c0', mb: 2 }}>Nhập admin token để tiếp tục (DEV: <code>dev-admin</code>).</Box>
          {err && <Box sx={{ fontSize: 12, color: '#ff5470', mb: 1 }}>{err}</Box>}
          <Box component="input" defaultValue={token} placeholder="admin token" id="admtok" sx={fieldSx} />
          <Button fullWidth variant="contained" sx={{ mt: 1.5, bgcolor: '#fe2d9b' }}
            onClick={() => { const v = document.getElementById('admtok').value.trim(); adminToken.set(v); setToken(v); setModels(null); setErr(''); loadModels(); }}>
            Vào editor
          </Button>
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', bgcolor: '#0b0e14', color: '#e9eef8', fontFamily: 'ui-sans-serif,system-ui,sans-serif' }}>
      {/* toolbar */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 2, py: 1.25, borderBottom: '1px solid #1e2836', flexWrap: 'wrap' }}>
        <Box sx={{ fontWeight: 700 }}>Story Editor</Box>
        <Box component="select" value={modelId ?? ''} onChange={(e) => setModelId(Number(e.target.value))} sx={{ ...fieldSx, width: 'auto' }}>
          {(models || []).map((m) => <option key={m.id} value={m.id}>{m.displayName || m.code}</option>)}
        </Box>
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          {(story?.chapters || []).map((c, i) => (
            <Button key={i} size="small" onClick={() => { setChIdx(i); setSel(null); }}
              sx={{ minWidth: 0, px: 1.25, color: i === chIdx ? '#fff' : '#8aa0c0', bgcolor: i === chIdx ? '#fe2d9b' : 'transparent', '&:hover': { bgcolor: i === chIdx ? '#fe2d9b' : '#1a2130' } }}>
              Ch{c.idx}
            </Button>
          ))}
        </Box>
        <Box sx={{ flex: 1 }} />
        <Button size="small" onClick={addScene} sx={{ color: '#e9eef8', border: '1px solid #2a3547' }}>+ Scene</Button>
        <Button size="small" onClick={deleteSel} disabled={!sel} sx={{ color: '#ff8fa3', border: '1px solid #2a3547' }}>Xoá</Button>
        <Button size="small" onClick={genPlayerMap} sx={{ color: '#2dd4a7', border: '1px solid #2a3547' }}>Sinh bản đồ người-chơi</Button>
        <Button size="small" variant="contained" onClick={save} disabled={!dirty} sx={{ bgcolor: '#fe2d9b' }}>Lưu story</Button>
        <Box sx={{ fontSize: 12, color: status.includes('✓') ? '#5be6a0' : '#ffd27a', minWidth: 70 }}>{status}</Box>
      </Box>
      {err && <Box sx={{ px: 2, py: 0.75, bgcolor: '#3a0f1c', color: '#ff8fa3', fontSize: 12.5 }}>{err}</Box>}

      {/* main */}
      <Box sx={{ flex: 1, display: 'flex', minHeight: 0 }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          {chapter && <FlowEditor key={`${modelId}-${chIdx}`} chapter={chapter} onChange={updateChapter} onSelect={setSel} />}
        </Box>
        {/* inspector */}
        <Box sx={{ width: 320, borderLeft: '1px solid #1e2836', bgcolor: '#11161f', p: 2, overflowY: 'auto' }}>
          <Inspector sel={sel} chapter={chapter} sceneCodes={sceneCodes}
            patchScene={patchScene} patchChoice={patchChoice} updateChapter={updateChapter} />
        </Box>
      </Box>
    </Box>
  );
}

function Inspector({ sel, chapter, sceneCodes, patchScene, patchChoice, updateChapter }) {
  if (!sel || !chapter) {
    return <Box sx={{ fontSize: 12.5, color: '#6b7a94' }}>Chọn một node (scene) hoặc cạnh (choice) để sửa. Kéo từ chấm phải của node sang node khác để nối.</Box>;
  }
  const scene = chapter.scenes.find((s) => s.code === sel.code);
  if (!scene) return null;

  if (sel.kind === 'scene') {
    return (
      <Box>
        <Box sx={{ fontSize: 13, fontWeight: 700, mb: 0.5 }}>Scene</Box>
        <Label>Code</Label>
        <Box component="input" value={scene.code} onChange={(e) => patchScene(scene.code, { code: e.target.value })} sx={fieldSx} />
        <Label>Type</Label>
        <Box component="select" value={scene.type} onChange={(e) => patchScene(scene.code, { type: e.target.value })} sx={fieldSx}>
          {SCENE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
        </Box>
        <Label>Video</Label>
        <Box component="select" value={scene.video || ''} onChange={(e) => patchScene(scene.code, { video: e.target.value || undefined })} sx={fieldSx}>
          <option value="">(none)</option>
          {(chapter.videos || []).map((v) => <option key={v.code} value={v.code}>{v.code} — {v.title}</option>)}
        </Box>
        {scene.type === 'linear' && (
          <>
            <Label>Next scene</Label>
            <Box component="select" value={scene.next || ''} onChange={(e) => patchScene(scene.code, { next: e.target.value })} sx={fieldSx}>
              <option value="">(chọn)</option>
              {sceneCodes.filter((c) => c !== scene.code).map((c) => <option key={c} value={c}>{c}</option>)}
            </Box>
          </>
        )}
        <Label>on_enter (effects JSON)</Label>
        <JsonField value={scene.on_enter} onApply={(o) => patchScene(scene.code, { on_enter: o })} />
        <Label>Entry của chapter</Label>
        <Button size="small" onClick={() => updateChapter({ ...chapter, entry: scene.code })}
          disabled={chapter.entry === scene.code}
          sx={{ color: '#2dd4a7', border: '1px solid #2a3547', mt: 0.5 }}>
          {chapter.entry === scene.code ? '✓ Là entry' : 'Đặt làm entry'}
        </Button>
      </Box>
    );
  }

  // choice
  const c = scene.choices?.[sel.choiceIdx];
  if (!c) return null;
  return (
    <Box>
      <Box sx={{ fontSize: 13, fontWeight: 700, mb: 0.5 }}>Choice · {scene.code}</Box>
      <Label>Label</Label>
      <Box component="input" value={c.label || ''} onChange={(e) => patchChoice(scene.code, sel.choiceIdx, { label: e.target.value })} sx={fieldSx} />
      <Label>Next scene</Label>
      <Box component="select" value={c.next || ''} onChange={(e) => patchChoice(scene.code, sel.choiceIdx, { next: e.target.value })} sx={fieldSx}>
        <option value="">(chọn)</option>
        {sceneCodes.map((sc) => <option key={sc} value={sc}>{sc}</option>)}
      </Box>
      <Label>condition (JSON)</Label>
      <JsonField value={c.condition} onApply={(o) => patchChoice(scene.code, sel.choiceIdx, { condition: o })} />
      <Label>effects (JSON)</Label>
      <JsonField value={c.effects} onApply={(o) => patchChoice(scene.code, sel.choiceIdx, { effects: o })} />
      <Label>timer_ms</Label>
      <Box component="input" type="number" value={c.timer_ms || ''} onChange={(e) => patchChoice(scene.code, sel.choiceIdx, { timer_ms: e.target.value ? Number(e.target.value) : undefined })} sx={fieldSx} />
    </Box>
  );
}
