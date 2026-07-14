// Màn bản đồ chapter — Figma "Don't Go Home Yet" (node 191:2202). DATA-DRIVEN: nodes (id +
// vị trí + poster) và edges (đồ thị nhánh thật của story) lấy từ GET /api/chapters/{id}/map;
// canvas rộng cuộn ngang, stage 16:9. Cạnh nối vẽ đứt nét bằng SVG theo `edges` (không suy
// diễn từ thứ tự x); badge START/FINISH + card video là node dữ liệu.
import { useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';

// Models
import { api } from '@models/game';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import VideoNode from '../videoNode';
import MapBadge from '../mapBadge';
import BackButton from '../backButton';

const CH = '/chapter'; // tái dùng bg.png + lock-heart.png

const STAGE_SX = {
  position: 'relative',
  width: 'min(100vw, calc(100vh * 16 / 9))',
  aspectRatio: '16 / 9',
  maxHeight: '100vh',
  overflow: 'hidden',
  containerType: 'size',
  background: `url('${CH}/bg.png') center / cover no-repeat`,
};

export default function ChapterMapScreen({ chapterId, onBack, onPlayVideo }) {
  const { t } = useTranslation();
  const [map, setMap] = useState(null);

  useEffect(() => {
    api.chapterMap(chapterId).then(setMap).catch(() => {});
  }, [chapterId]);

  const W = map?.width || 2000;
  const H = map?.height || 760;
  const nodes = map?.nodes ?? [];
  const edges = map?.edges ?? [];
  const byId = Object.fromEntries(nodes.map((n) => [n.id, n]));
  const center = (n) => ({ cx: n.x + n.w / 2, cy: n.y + n.h / 2 });
  const pct = (v, total) => `${(v / total) * 100}%`;

  // Cha trực tiếp của mỗi node (theo edges from→to).
  const preds = {};
  edges.forEach((e) => {
    (preds[e.to] = preds[e.to] || []).push(e.from);
  });
  // Video cha GẦN NHẤT đã mở khóa của 1 node (đi ngược theo edges). Bấm node chưa mở →
  // play video này (video trước đó cần xem để có lựa chọn dẫn tới node đang khóa).
  const nearestPlayable = (startId) => {
    const seen = new Set();
    const queue = [...(preds[startId] || [])];
    while (queue.length) {
      const id = queue.shift();
      if (seen.has(id)) continue;
      seen.add(id);
      const nd = byId[id];
      if (!nd) continue;
      if (nd.kind === 'video' && !nd.locked && nd.scene) return nd;
      (preds[id] || []).forEach((p) => queue.push(p));
    }
    return null;
  };
  const handlePlay = (n) => {
    if (!n.locked) {
      onPlayVideo(n.scene);
      return;
    }
    const prev = nearestPlayable(n.id); // node khóa → play video cha đã mở gần nhất
    if (prev) onPlayVideo(prev.scene);
  };

  // Cạnh nối theo `edges` từ server (đồ thị nhánh thật của story) — nối tâm node `from` → `to`.
  // `chosen` = nhánh user đã đi qua (server tính) → vẽ đường liền trắng + icon kim cương.
  const links = edges
    .map((e, i) => ({ a: byId[e.from], b: byId[e.to], key: `e${i}`, chosen: !!e.chosen }))
    .filter(({ a, b }) => a && b);

  return (
    <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#000' }}>
      <Box sx={STAGE_SX}>
        {/* ===== Canvas cuộn ngang ===== */}
        <Box sx={{ position: 'absolute', inset: 0, overflowX: 'auto', overflowY: 'hidden', WebkitOverflowScrolling: 'touch', '&::-webkit-scrollbar': { display: 'none' }, scrollbarWidth: 'none' }}>
          <Box sx={{ position: 'relative', height: '100%', aspectRatio: `${W} / ${H}` }}>
            {/* Đường nối đứt nét */}
            <Box
              component="svg"
              viewBox={`0 0 ${W} ${H}`}
              preserveAspectRatio="none"
              sx={{
                position: 'absolute',
                inset: 0,
                width: '100%',
                height: '100%',
                pointerEvents: 'none',
                // Mặc định: đứt nét mờ. Nhánh đã đi (.edge-chosen): liền, trắng đậm, dày hơn.
                '& line': { stroke: 'rgba(255,255,255,0.85)', strokeWidth: 3, strokeDasharray: '6 7', strokeLinecap: 'round' },
                '& line.edge-chosen': { stroke: '#fff', strokeWidth: 4, strokeDasharray: 'none' },
              }}
            >
              {/* Cạnh nối theo đồ thị story (edges): tâm from → tâm to */}
              {links.map(({ a, b, key, chosen }) => (
                <line
                  key={key}
                  className={chosen ? 'edge-chosen' : undefined}
                  x1={center(a).cx}
                  y1={center(a).cy}
                  x2={center(b).cx}
                  y2={center(b).cy}
                />
              ))}
            </Box>

            {/* Icon ở giữa mỗi đường nối (midpoint của edge).
                Bỏ qua cạnh chạm node start/finish — chỉ hiện giữa các node video.
                Ưu tiên: nhánh đã đi → kim cương hồng; else node hotspot → marker ❗; else ổ khóa. */}
            {links.map(({ a, b, key, chosen }) => {
              if (a.kind === 'start' || a.kind === 'finish' || b.kind === 'start' || b.kind === 'finish') return null;
              const isHotspot = a.hotspot || b.hotspot;
              const src = chosen
                ? '/icon/diamond-chosen.png'
                : isHotspot
                  ? '/icon/hotspot-marker.png'
                  : '/icon/heart-lock.png';
              const mx = (center(a).cx + center(b).cx) / 2;
              const my = (center(a).cy + center(b).cy) / 2;
              return (
                <Box
                  key={`icon-${key}`}
                  component="img"
                  src={src}
                  alt=""
                  aria-hidden
                  sx={{
                    position: 'absolute',
                    left: pct(mx, W),
                    top: pct(my, H),
                    width: chosen ? '2.4cqw' : '3.6cqw',
                    height: 'auto',
                    transform: 'translate(-50%, -50%)',
                    objectFit: 'contain',
                    pointerEvents: 'none',
                    filter: 'drop-shadow(0 0.3cqw 0.6cqw rgba(0,0,0,0.35))',
                  }}
                />
              );
            })}

            {/* Node */}
            {nodes.map((n, i) => {
              const style = { left: pct(n.x, W), top: pct(n.y, H), width: pct(n.w, W), height: pct(n.h, H) };
              if (n.kind === 'video') {
                return (
                  <VideoNode
                    key={i}
                    index={i}
                    title={n.title}
                    posterUrl={n.posterUrl}
                    style={style}
                    onPlay={() => handlePlay(n)}
                  />
                );
              }
              if (n.kind === 'start') {
                return <MapBadge key={i} id="map-start" label={t('map_w1_start', 'START')} style={style} />;
              }
              if (n.kind === 'finish') {
                return <MapBadge key={i} id="map-finish" label={t('map_w1_finish', 'FINISH')} style={style} />;
              }
              // lock
              return (
                <Box
                  key={i}
                  component="img"
                  src={`${CH}/lock-heart.png`}
                  alt=""
                  aria-hidden
                  sx={{ position: 'absolute', ...style, objectFit: 'contain', filter: 'drop-shadow(0 2px 5px rgba(0,0,0,0.35))' }}
                />
              );
            })}
          </Box>
        </Box>

        {/* ===== Header CHAPTER + số ===== */}
        <Box
          component="span"
          sx={{
            position: 'absolute', left: '63.7cqw', top: '5.5cqh', fontFamily: "'Roboto', sans-serif", fontWeight: 700, fontSize: '5.2cqw', lineHeight: 1,
            whiteSpace: 'nowrap', letterSpacing: '0.05cqw', pointerEvents: 'none',
            background: 'linear-gradient(180deg, rgba(255,255,255,0.85) 50%, rgba(170,170,170,0.45) 110%)',
            WebkitBackgroundClip: 'text', backgroundClip: 'text', color: 'transparent',
          }}
        >
          {t('map_w1_chapter', 'CHAPTER')}
        </Box>
        <Box
          component="span"
          sx={{ position: 'absolute', left: '89cqw', top: '-1cqh', fontFamily: "'Roboto', sans-serif", fontWeight: 900, fontSize: '17cqw', lineHeight: 1, color: '#fff', opacity: 0.92, textShadow: '0 0.4cqw 1cqw rgba(0,0,0,0.18)', pointerEvents: 'none' }}
        >
          {chapterId}
        </Box>
        <Box aria-hidden sx={{ position: 'absolute', left: '56.8cqw', top: '15.4cqh', width: '32.9cqw', height: '0.4cqh', opacity: 0.5, background: 'linear-gradient(90deg, rgba(255,255,255,0) 4%, #fff 30%)', pointerEvents: 'none' }} />
        <Box component="span" sx={{ position: 'absolute', left: '76.7cqw', top: '17.4cqh', fontFamily: "'Roboto', sans-serif", fontWeight: 700, fontSize: '2.5cqw', lineHeight: 1, color: 'rgba(255,255,255,0.9)', whiteSpace: 'nowrap', pointerEvents: 'none' }}>
          {t('map_w1_subtitle', 'Your Number')}
        </Box>

        {/* Back */}
        <BackButton id="chapter-map-back-btn" onClick={onBack} />
      </Box>
    </Box>
  );
}
