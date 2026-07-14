// <PlayerCore> = <VideoSurface> + (<HotspotLayer> | <ChoiceOverlay>) + <BranchPreloader>
//  - HotspotLayer: choice có toạ độ `hotspot` → vùng bấm trên khung (cửa sáng + marker ❗),
//    theo Figma "Frame 57" (node 170:1026).
//  - ChoiceOverlay: choice không hotspot → thẻ câu hỏi góc dưới-trái, theo "Frame 68" (node 188:140).
import { useEffect, useRef, useState } from 'react';

// MUI
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import HotspotArea from '@components/hotspotArea';
import BackButton from '../backButton';

// Khung 16:9 letterbox + container-query để đặt UI theo tỉ lệ khung 1920×1080
// của design (cqw = 1% chiều rộng khung, cqh = 1% chiều cao).
const STAGE_SX = {
  position: 'relative',
  width: 'min(100cqw, calc(100cqh * 16 / 9))',
  aspectRatio: '16 / 9',
  overflow: 'hidden',
  containerType: 'size',
};

const CARD_GRADIENT = 'linear-gradient(105deg, #A4DAFF 1.64%, #FFCDF6 100%)';

export default function PlayerCore({ scene, busy, onLinearEnded, onEndingEnded, onChoose, onClose }) {
  const [videoEnded, setVideoEnded] = useState(false);

  useEffect(() => setVideoEnded(false), [scene.scene.id]);

  const showChoices = scene.scene.type === 'choice' && videoEnded && !busy;
  // Hotspot khi MỌI choice hiển thị đều có toạ độ hotspot; ngược lại → thẻ câu hỏi.
  const useHotspots =
    showChoices && scene.choices.length > 0 && scene.choices.every((c) => c.hotspot);

  return (
    <Box
      sx={{
        position: 'absolute',
        inset: 0,
        bgcolor: '#000',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Box sx={STAGE_SX}>
        <VideoSurface
          key={scene.scene.id}
          src={scene.videoUrl}
          onEnded={() => {
            setVideoEnded(true);
            if (scene.scene.type === 'linear') onLinearEnded();
            else if (scene.scene.type === 'ending') onEndingEnded?.();
          }}
        />
        {useHotspots && (
          <HotspotLayer key={scene.scene.id} choices={scene.choices} onChoose={onChoose} />
        )}
        {showChoices && !useHotspots && (
          <ChoiceOverlay key={scene.scene.id} choices={scene.choices} onChoose={onChoose} />
        )}
        <BranchPreloader choices={scene.choices} />
        {/* Đóng video → quay về bản đồ chapter */}
        {onClose && <BackButton id="player-back-btn" onClick={onClose} />}
      </Box>
      {busy && (
        <Box
          sx={{
            position: 'absolute',
            bottom: 18,
            right: 22,
            color: 'primary.main',
            animation: 'pulse 0.8s infinite alternate',
            '@keyframes pulse': { from: { opacity: 0.3 }, to: { opacity: 1 } },
          }}
        >
          ●
        </Box>
      )}
    </Box>
  );
}

function VideoSurface({ src, onEnded }) {
  const ref = useRef(null);
  const firedRef = useRef(false);

  // Không có video (scene "câm") → coi như clip 1.5s
  useEffect(() => {
    if (src) return;
    const t = setTimeout(onEnded, 1500);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [src]);

  const fire = () => {
    if (!firedRef.current) {
      firedRef.current = true;
      onEnded();
    }
  };

  if (!src) return <Box sx={{ width: '100%', height: '100%', bgcolor: '#000' }} />;
  return (
    <Box
      component="video"
      ref={ref}
      src={src}
      autoPlay
      playsInline
      onEnded={fire}
      onError={fire} // Clip hỏng/thiếu: đừng kẹt người chơi — coi như đã phát xong
      sx={{ width: '100%', height: '100%', objectFit: 'contain' }}
    />
  );
}

// useChoiceTimer — đếm giờ dùng chung: hết giờ tự chọn default (hoặc choice đầu).
// `choose` phải tự guard idempotent.
function useChoiceTimer(choices, choose) {
  const timerMs = choices.find((c) => c.timerMs != null)?.timerMs ?? null;
  const [remaining, setRemaining] = useState(timerMs);

  useEffect(() => {
    if (timerMs == null) return;
    const startedAt = Date.now();
    const iv = setInterval(() => {
      const left = timerMs - (Date.now() - startedAt);
      setRemaining(Math.max(0, left));
      if (left <= 0) {
        clearInterval(iv);
        const fallback = choices.find((c) => c.isDefault) ?? choices[0];
        if (fallback) choose(fallback.id);
      }
    }, 100);
    return () => clearInterval(iv);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timerMs]);

  return { timerMs, remaining };
}

function TimerBar({ timerMs, remaining, sx }) {
  if (timerMs == null || remaining == null) return null;
  return (
    <Box
      sx={{
        width: '23.6cqw',
        height: '0.9cqh',
        bgcolor: 'rgba(0,0,0,0.35)',
        borderRadius: 999,
        overflow: 'hidden',
        ...sx,
      }}
    >
      <Box
        sx={{
          height: '100%',
          width: `${(remaining / timerMs) * 100}%`,
          background: 'linear-gradient(90deg, #A4DAFF, #FFCDF6)',
          transition: 'width 0.1s linear',
        }}
      />
    </Box>
  );
}

// HotspotLayer — vùng bấm định vị trên khung video (màn "Tương tác", Frame 57):
// mỗi choice là một vùng cửa phát sáng hồng (Vector 7) + marker ❗ diamond ở góc
// trên-trái (Group 10). Toạ độ 0..1 do server trả (server-authoritative).
function HotspotLayer({ choices, onChoose }) {
  const chosenRef = useRef(false);
  const choose = (id) => {
    if (chosenRef.current) return;
    chosenRef.current = true;
    onChoose(id);
  };
  const { timerMs, remaining } = useChoiceTimer(choices, choose);

  return (
    <Box sx={{ position: 'absolute', inset: 0 }}>
      {choices.map((c) => (
        <HotspotArea
          key={c.id}
          id={`player-hotspot-${c.id}`}
          label={c.label}
          onClick={() => choose(c.id)}
          left={`${c.hotspot.x * 100}%`}
          top={`${c.hotspot.y * 100}%`}
          width={`${c.hotspot.w * 100}%`}
          height={`${c.hotspot.h * 100}%`}
          perspective={c.hotspot.perspective}
          tilt={c.hotspot.tilt}
        />
      ))}
      {/* Thanh đếm giờ (nếu có) — góc dưới-trái, đồng bộ với thẻ câu hỏi */}
      <TimerBar timerMs={timerMs} remaining={remaining} sx={{ position: 'absolute', left: '11.35cqw', bottom: '6cqh' }} />
    </Box>
  );
}

// ChoiceOverlay — các lựa chọn hiện dạng THẺ CÂU HỎI xếp GÓC DƯỚI-TRÁI khung hình
// (Frame 68, node 188:140). Có timer thì tự chọn default khi hết giờ.
function ChoiceOverlay({ choices, onChoose }) {
  const { t } = useTranslation();
  const chosenRef = useRef(false);
  const choose = (id) => {
    if (chosenRef.current) return;
    chosenRef.current = true;
    onChoose(id);
  };
  const { timerMs, remaining } = useChoiceTimer(choices, choose);

  return (
    <Stack
      sx={{
        position: 'absolute',
        left: 0,
        right: 0,
        bottom: '6cqh',
        alignItems: 'center', // luôn căn giữa theo chiều ngang
        gap: '1.6cqh',
      }}
    >
      <TimerBar timerMs={timerMs} remaining={remaining} />
      {/* Các thẻ trả lời xếp theo CHIỀU NGANG, căn giữa */}
      <Stack
        direction="row"
        sx={{
          gap: '2cqw',
          justifyContent: 'center',
          alignItems: 'stretch',
          flexWrap: 'wrap',
          maxWidth: '92cqw',
          px: '2cqw',
        }}
      >
        {choices.map((c) => (
          <QuestionCard
            key={c.id}
            id={`player-choice-${c.id}-btn`}
            label={c.label}
            isDefault={c.isDefault}
            defaultTag={t('player_w1_default_tag', 'tự chọn khi hết giờ')}
            onClick={() => choose(c.id)}
          />
        ))}
      </Stack>
    </Stack>
  );
}

function QuestionCard({ id, label, isDefault, defaultTag, onClick }) {
  return (
    <Box
      id={id}
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onClick();
      }}
      sx={{
        position: 'relative',
        minWidth: '23.6cqw', // 453/1920
        maxWidth: '52cqw',
        px: '2.2cqw',
        py: '1.9cqh',
        cursor: 'pointer',
        background: CARD_GRADIENT,
        border: '2px solid #fff',
        // Bo góc bất đối xứng đặc trưng: chỉ top-left + bottom-right
        borderRadius: '1.7cqw 0 1.7cqw 0',
        boxShadow: '0 0.5cqh 2cqh rgba(0,0,0,0.35)',
        overflow: 'hidden',
        transition: 'transform 0.12s ease, filter 0.12s ease',
        '&:hover': { transform: 'translateY(-2px)', filter: 'brightness(1.04)' },
        '&:focus-visible': { outline: '2px solid #fff', outlineOffset: 2 },
      }}
    >
      <Box
        aria-hidden
        component="span"
        sx={{ position: 'absolute', top: '10%', right: '6%', color: 'rgba(255,255,255,0.55)', fontSize: '1.4cqw' }}
      >
        ✦
      </Box>
      <Box
        aria-hidden
        component="span"
        sx={{ position: 'absolute', bottom: '12%', left: '5%', color: 'rgba(255,255,255,0.45)', fontSize: '1cqw' }}
      >
        ✦
      </Box>
      <Typography
        sx={{
          fontFamily: "'Mochiy Pop One', sans-serif",
          fontSize: '1.55cqw',
          lineHeight: 1.25,
          color: '#111',
          textAlign: 'center',
        }}
      >
        {label}
      </Typography>
      {isDefault && (
        <Typography sx={{ mt: '0.4cqh', fontSize: '1cqw', color: 'rgba(0,0,0,0.55)', textAlign: 'center' }}>
          {defaultTag}
        </Typography>
      )}
    </Box>
  );
}

// Preload ẩn đoạn đầu clip của từng nhánh → bấm choice là phát ngay, không loading.
function BranchPreloader({ choices }) {
  return (
    <Box sx={{ display: 'none' }} aria-hidden>
      {choices
        .filter((c) => c.preloadUrl)
        .map((c) => (
          <video key={c.id} src={c.preloadUrl} preload="auto" muted />
        ))}
    </Box>
  );
}
