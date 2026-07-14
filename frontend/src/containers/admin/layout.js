// Layout helpers cho story editor + sinh bản đồ người-chơi.
//  - autoLayout: xếp scene theo độ sâu BFS từ entry (fallback khi chưa kéo tay).
//  - buildPlayerMap: dựng map_json (start + video card + finish + edges) TỪ story graph,
//    dùng ĐÚNG vị trí admin đã kéo trong editor (chapter.map.editor) → "admin sắp = player thấy".

// Cạnh đi ra của 1 scene (linear → next; choice → mọi choice.next).
const outsOf = (sc) => (sc.type === 'choice' ? (sc.choices || []).map((c) => c.next) : sc.next ? [sc.next] : []);

export function autoLayout(chapter) {
  const scenes = chapter.scenes || [];
  const codes = new Set(scenes.map((s) => s.code));
  const depth = {};
  const entry = chapter.entry && codes.has(chapter.entry) ? chapter.entry : scenes[0]?.code;
  const queue = [];
  if (entry) { depth[entry] = 0; queue.push(entry); }
  while (queue.length) {
    const cur = queue.shift();
    const sc = scenes.find((s) => s.code === cur);
    if (!sc) continue;
    for (const nx of outsOf(sc)) {
      if (codes.has(nx) && depth[nx] === undefined) { depth[nx] = depth[cur] + 1; queue.push(nx); }
    }
  }
  const rowInCol = {};
  const pos = {};
  const maxDepth = Math.max(0, ...Object.values(depth));
  scenes.forEach((s) => {
    const d = depth[s.code] ?? maxDepth + 1; // node rời → cột cuối
    rowInCol[d] = rowInCol[d] || 0;
    pos[s.code] = { x: 60 + d * 260, y: 40 + rowInCol[d] * 130 };
    rowInCol[d] += 1;
  });
  return pos;
}

// buildPlayerMap — trả object map_json (khớp director.rawMap: width/height/nodes/edges).
// Giữ lại chapter.map.editor để lần sinh sau vẫn dùng vị trí đã kéo.
export function buildPlayerMap(chapter) {
  const S = 1.7; // scale vị trí editor (node nhỏ) → card 240×135 cho thoáng
  const VW = 240, VH = 135, DW = 150, DH = 150, PAD = 80;
  const saved = chapter.map?.editor || {};
  const auto = autoLayout(chapter);
  const posOf = (code) => saved[code] || auto[code] || { x: 0, y: 0 };
  const videoByCode = {};
  (chapter.videos || []).forEach((v) => { videoByCode[v.code] = v; });

  const nodes = [];
  (chapter.scenes || []).forEach((sc) => {
    const p = posOf(sc.code);
    const x = Math.round(p.x * S), y = Math.round(p.y * S);
    if (sc.type === 'ending') {
      nodes.push({ id: sc.code, kind: 'finish', x, y, w: DW, h: DH });
    } else {
      const v = videoByCode[sc.video];
      const n = { id: sc.code, kind: 'video', title: v?.title || sc.code, x, y, w: VW, h: VH };
      if (v?.poster) n.poster = v.poster;
      nodes.push(n);
    }
  });

  const edges = [];
  const entry = chapter.entry;
  if (entry && (chapter.scenes || []).some((s) => s.code === entry)) {
    const p = posOf(entry);
    nodes.unshift({ id: '__start', kind: 'start', x: Math.round(p.x * S) - 210, y: Math.round(p.y * S) - (DH - VH) / 2, w: DW, h: DH });
    edges.push({ from: '__start', to: entry });
  }
  (chapter.scenes || []).forEach((sc) => {
    for (const to of outsOf(sc)) if (to) edges.push({ from: sc.code, to });
  });

  // dồn về góc (PAD, PAD) + tính khung.
  let minX = Infinity, minY = Infinity, maxX = 0, maxY = 0;
  nodes.forEach((n) => { minX = Math.min(minX, n.x); minY = Math.min(minY, n.y); maxX = Math.max(maxX, n.x + n.w); maxY = Math.max(maxY, n.y + n.h); });
  const dx = PAD - minX, dy = PAD - minY;
  nodes.forEach((n) => { n.x += dx; n.y += dy; });

  return { width: maxX + dx + PAD, height: maxY + dy + PAD, nodes, edges, editor: saved };
}
