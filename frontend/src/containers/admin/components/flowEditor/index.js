// FlowEditor — trình sửa đồ thị story bằng React Flow cho 1 chapter.
// Nguồn chân lý = ChapterDef (StoryFile). Node = scene, cạnh = next (linear) / choice.
// Kéo-thả lưu vị trí vào chapter.map.editor[code]; sửa logic qua inspector; nối handle
// tạo cạnh (choice hoặc set next). onChange trả về ChapterDef mới cho AdminApp gộp + lưu.
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ReactFlow, Background, Controls, MiniMap, Handle, Position,
  useNodesState, useEdgesState,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import Box from '@mui/material/Box';

import { autoLayout } from '../../layout';

const TYPE_COLOR = { linear: '#3a86ff', choice: '#fe2d9b', ending: '#ffb703', entry: '#2dd4a7' };

// ---------- custom node ----------
function SceneNode({ data, selected }) {
  const c = data.isEntry ? TYPE_COLOR.entry : (TYPE_COLOR[data.type] || '#888');
  return (
    <Box sx={{
      minWidth: 150, borderRadius: 1.5, bgcolor: '#161b26', color: '#e9eef8',
      border: `2px solid ${selected ? '#fff' : c}`, boxShadow: selected ? '0 0 0 3px rgba(255,255,255,.25)' : '0 4px 12px rgba(0,0,0,.4)',
      overflow: 'hidden', fontFamily: 'ui-sans-serif,system-ui,sans-serif',
    }}>
      <Handle type="target" position={Position.Left} style={{ background: c, width: 9, height: 9 }} />
      <Box sx={{ px: 1, py: 0.5, bgcolor: c, color: '#0b0e14', fontSize: 10, fontWeight: 800, letterSpacing: '.4px', textTransform: 'uppercase', display: 'flex', justifyContent: 'space-between', gap: 1 }}>
        <span>{data.isEntry ? `▶ ${data.type}` : data.type}</span>
        {data.video ? <span style={{ opacity: 0.75, fontWeight: 600 }}>🎬</span> : null}
      </Box>
      <Box sx={{ px: 1, py: 0.75 }}>
        <Box sx={{ fontSize: 12, fontWeight: 700 }}>{data.code}</Box>
        {data.video ? <Box sx={{ fontSize: 10, color: '#8aa0c0', mt: 0.25 }}>{data.video}</Box> : null}
      </Box>
      <Handle type="source" position={Position.Right} style={{ background: c, width: 9, height: 9 }} />
    </Box>
  );
}
const nodeTypes = { scene: SceneNode };

// ---------- build nodes/edges từ chapter ----------
function buildGraph(chapter, positions) {
  const scenes = chapter.scenes || [];
  const nodes = scenes.map((s) => ({
    id: s.code,
    type: 'scene',
    position: positions[s.code] || { x: 60, y: 40 },
    data: { code: s.code, type: s.type, video: s.video, isEntry: s.code === chapter.entry },
  }));
  const edges = [];
  scenes.forEach((s) => {
    if (s.type === 'linear' && s.next) {
      edges.push({ id: `${s.code}->${s.next}`, source: s.code, target: s.next, animated: false,
        style: { stroke: '#5b6b86', strokeWidth: 2 } });
    } else if (s.type === 'choice') {
      (s.choices || []).forEach((c, i) => {
        edges.push({
          id: `${s.code}__c${i}`, source: s.code, target: c.next, label: c.label,
          data: { choiceIdx: i }, labelStyle: { fill: '#e9eef8', fontSize: 10, fontWeight: 600 },
          labelBgStyle: { fill: '#2a1830' }, labelBgPadding: [4, 2], labelBgBorderRadius: 4,
          style: { stroke: TYPE_COLOR.choice, strokeWidth: 2 },
        });
      });
    }
  });
  return { nodes, edges };
}

export default function FlowEditor({ chapter, onChange, onSelect }) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  // vị trí lưu ở chapter.map.editor (persist qua /content). Auto-layout nếu chưa có.
  const positions = useMemo(() => {
    const saved = chapter?.map?.editor || {};
    const auto = autoLayout(chapter || { scenes: [] });
    const out = {};
    (chapter?.scenes || []).forEach((s) => { out[s.code] = saved[s.code] || auto[s.code]; });
    return out;
  }, [chapter]);

  // rebuild khi đổi chapter / cấu trúc.
  const chapterSig = useMemo(
    () => JSON.stringify((chapter?.scenes || []).map((s) => [s.code, s.type, s.video, s.next, (s.choices || []).map((c) => [c.label, c.next])])) + chapter?.entry,
    [chapter],
  );
  useEffect(() => {
    const g = buildGraph(chapter || { scenes: [] }, positions);
    setNodes(g.nodes);
    setEdges(g.edges);
  }, [chapterSig]); // eslint-disable-line react-hooks/exhaustive-deps

  // kéo node → ghi vị trí vào chapter.map.editor
  const persistPositions = useCallback((nextNodes) => {
    const editor = {};
    nextNodes.forEach((n) => { editor[n.id] = { x: Math.round(n.position.x), y: Math.round(n.position.y) }; });
    const map = { ...(chapter.map || {}), editor };
    onChange({ ...chapter, map });
  }, [chapter, onChange]);

  const handleNodesChange = useCallback((changes) => {
    onNodesChange(changes);
    if (changes.some((c) => c.type === 'position' && c.dragging === false)) {
      setNodes((cur) => { persistPositions(cur); return cur; });
    }
  }, [onNodesChange, setNodes, persistPositions]);

  // nối handle → tạo cạnh: choice thì thêm choice; linear thì set next (thay thế).
  const onConnect = useCallback((conn) => {
    const { source, target } = conn;
    if (!source || !target) return;
    const scenes = chapter.scenes.map((s) => {
      if (s.code !== source) return s;
      if (s.type === 'choice') {
        return { ...s, choices: [...(s.choices || []), { label: 'Lựa chọn mới', next: target }] };
      }
      if (s.type === 'linear') return { ...s, next: target };
      return s; // ending: bỏ qua
    });
    onChange({ ...chapter, scenes });
  }, [chapter, onChange]);

  const onNodeClick = useCallback((_, n) => onSelect?.({ kind: 'scene', code: n.id }), [onSelect]);
  const onEdgeClick = useCallback((_, e) => {
    if (e.data?.choiceIdx != null) onSelect?.({ kind: 'choice', code: e.source, choiceIdx: e.data.choiceIdx });
    else onSelect?.({ kind: 'scene', code: e.source });
  }, [onSelect]);
  const onPaneClick = useCallback(() => onSelect?.(null), [onSelect]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      onNodesChange={handleNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      onNodeClick={onNodeClick}
      onEdgeClick={onEdgeClick}
      onPaneClick={onPaneClick}
      fitView
      proOptions={{ hideAttribution: true }}
      style={{ background: '#0b0e14' }}
    >
      <Background color="#1e2836" gap={22} />
      <Controls />
      <MiniMap pannable zoomable nodeColor={(n) => (n.data?.isEntry ? TYPE_COLOR.entry : TYPE_COLOR[n.data?.type] || '#888')} style={{ background: '#11161f' }} />
    </ReactFlow>
  );
}
