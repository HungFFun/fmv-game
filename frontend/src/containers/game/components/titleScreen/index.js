// Màn hình đầu — Figma "Don't Go Home Yet" (node 139:57).
// Dựng lại bằng các phần tử rời (không còn 1 ảnh phẳng + hotspot): nền + logo + 4 nút hình thoi
// (Story/Ranking/Album/Settings) + nút Continue + chân dung nhân vật cắt hình thoi.
// Stage giữ tỉ lệ 16:9 và là một CSS container (cqw/cqh) nên mọi kích thước/vị trí/chữ
// scale đồng nhất theo khung 1920×1080 của design.

// MUI
import Box from '@mui/material/Box';

// Hooks
import useTranslation from '@hooks/useTranslation';

const A = '/home'; // thư mục asset tải từ Figma

const STAGE_SX = {
  position: 'relative',
  width: 'min(100cqw, calc(100cqh * 16 / 9))',
  aspectRatio: '16 / 9',
  maxHeight: '100cqh',
  overflow: 'hidden',
  containerType: 'size',
  background: `url('${A}/bg.png') center / cover no-repeat`,
};

// Một nút hình thoi: ô vuông xoay 45° làm khung trắng, nội dung (icon + nhãn) để thẳng ở trên.
function DiamondButton({ id, icon, label, cx, cy, onClick }) {
  return (
    <Box
      component="button"
      id={id}
      type="button"
      aria-label={label}
      onClick={onClick}
      sx={{
        position: 'absolute',
        left: `${cx}cqw`,
        top: `${cy}cqh`,
        width: '7.7cqw',
        aspectRatio: '1',
        transform: 'translate(-50%, -50%)',
        p: 0,
        border: 'none',
        background: 'transparent',
        cursor: 'pointer',
        transition: 'transform 0.12s ease',
        '&:hover': { transform: 'translate(-50%, -50%) scale(1.06)' },
        '&:active': { transform: 'translate(-50%, -50%) scale(0.97)' },
        '&:focus-visible': { outline: 'none' },
        '&:focus-visible .diamond': { boxShadow: '0 0 0 0.3cqw #fff' },
      }}
    >
      {/* Khung thoi trắng */}
      <Box
        className="diamond"
        sx={{
          position: 'absolute',
          inset: 0,
          transform: 'rotate(45deg)',
          borderRadius: '0.25cqw',
          background: 'linear-gradient(225deg, rgba(243,246,250,0.95) 0%, rgba(228,221,230,0.95) 100%)',
          border: '0.15cqw solid rgba(255,255,255,0.7)',
          boxShadow: '0 0.4cqw 1cqw rgba(40,30,60,0.25), inset 0.2cqw 0.12cqw 0.2cqw rgba(255,255,255,0.85)',
        }}
      />
      {/* Nội dung để thẳng (không xoay) */}
      <Box
        sx={{
          position: 'absolute',
          inset: 0,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '0.3cqw',
        }}
      >
        <Box component="img" src={icon} alt="" sx={{ width: '2.6cqw', height: '2.6cqw' }} />
        <Box
          component="span"
          sx={{
            fontFamily: "'Roboto', sans-serif",
            fontWeight: 700,
            fontSize: '1.3cqw',
            lineHeight: 1,
            color: '#203272',
            whiteSpace: 'nowrap',
          }}
        >
          {label}
        </Box>
      </Box>
    </Box>
  );
}

export default function TitleScreen({ onStart, onOpenGallery, onComingSoon }) {
  const { t } = useTranslation();

  const buttons = [
    { id: 'title-story-btn', icon: `${A}/icon-story.svg`, label: t('title_w1_story', 'Story'), cx: 12.1, cy: 61.3, onClick: onStart },
    { id: 'title-ranking-btn', icon: `${A}/icon-ranking.svg`, label: t('title_w1_ranking', 'Ranking'), cx: 21.4, cy: 66.5, onClick: () => onComingSoon(t('title_w1_ranking', 'Ranking')) },
    { id: 'title-album-btn', icon: `${A}/icon-album.svg`, label: t('title_w1_album', 'Album'), cx: 30.6, cy: 61.3, onClick: onOpenGallery },
    { id: 'title-settings-btn', icon: `${A}/icon-settings.svg`, label: t('title_w1_settings', 'Settings'), cx: 39.7, cy: 66.5, onClick: () => onComingSoon(t('title_w1_settings', 'Settings')) },
  ];

  return (
    <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#000' }}>
      <Box sx={STAGE_SX}>
        {/* Chân dung nhân vật — khung thoi trắng, ảnh để thẳng bên trong */}
        <Box
          sx={{
            position: 'absolute',
            left: '72.4cqw',
            top: '48.4cqh',
            width: '50.7cqw',
            aspectRatio: '1',
            transform: 'translate(-50%, -50%)',
            clipPath: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',
            background: '#fff',
            filter: 'drop-shadow(0 0 2cqw rgba(255,255,255,0.45))',
          }}
        >
          <Box
            sx={{
              position: 'absolute',
              inset: '1cqw',
              clipPath: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',
              background: `url('${A}/character.png') 50% 14% / cover no-repeat`,
            }}
          />
        </Box>

        {/* Logo tiêu đề */}
        <Box
          aria-hidden
          sx={{
            position: 'absolute',
            left: '3.6cqw',
            top: '-8.7cqh',
            width: '44.6cqw',
            height: '79.3cqh',
            background: `url('${A}/logo.png') center / contain no-repeat`,
            pointerEvents: 'none',
          }}
        />

        {/* Huy hiệu DEMO */}
        <Box
          aria-hidden
          sx={{
            position: 'absolute',
            left: '9.6cqw',
            top: '6cqh',
            transform: 'rotate(-15deg)',
            px: '1.4cqw',
            py: '0.4cqw',
            borderRadius: '999px',
            background: 'linear-gradient(121deg, #d0d6e5 10%, #afbde0 92%)',
            boxShadow: 'inset 0.2cqw 0.13cqw 0.18cqw rgba(255,255,255,0.8)',
            pointerEvents: 'none',
          }}
        >
          <Box
            component="span"
            sx={{
              fontFamily: "'Puffin Display Soft', 'Roboto', sans-serif",
              fontStyle: 'italic',
              fontWeight: 800,
              fontSize: '2.7cqw',
              lineHeight: 1,
              letterSpacing: '0.05cqw',
              background: 'linear-gradient(172deg, #0dd0a7 20%, #1957c6 79%)',
              WebkitBackgroundClip: 'text',
              backgroundClip: 'text',
              color: 'transparent',
              textShadow: '0.1cqw 0.1cqw 0.02cqw rgba(0,0,0,0.35)',
            }}
          >
            DEMO
          </Box>
        </Box>

        {/* 4 nút hình thoi */}
        {buttons.map((b) => (
          <DiamondButton key={b.id} {...b} />
        ))}

        {/* Nút Continue — khung ngoài hồng nhạt trong suốt bọc nền hồng đặc (đúng node 156:460) */}
        <Box
          component="button"
          id="title-start-btn"
          type="button"
          aria-label={t('title_w1_continue', 'Continue')}
          onClick={onStart}
          sx={{
            position: 'absolute',
            left: '14.1cqw',
            top: '79.7cqh',
            width: '23.7cqw',
            height: '12cqh',
            p: 0,
            border: 'none',
            borderRadius: '0.42cqw',
            background: 'rgba(255,221,253,0.5)',
            overflow: 'hidden',
            cursor: 'pointer',
            transition: 'transform 0.12s ease, filter 0.12s ease',
            '&:hover': { filter: 'brightness(1.06)', transform: 'scale(1.03)' },
            '&:active': { transform: 'scale(0.98)' },
            '&:focus-visible': { outline: '0.2cqw solid #fff', outlineOffset: '0.2cqw' },
          }}
        >
          {/* Nền hồng đặc + shadow kép */}
          <Box
            aria-hidden
            sx={{
              position: 'absolute',
              inset: '0.42cqw',
              borderRadius: '0.2cqw',
              background: '#fe2d9b',
              boxShadow: '-0.05cqw -0.05cqw 0.17cqw rgba(0,0,0,0.5), 0.05cqw 0.05cqw 0.17cqw rgba(0,0,0,0.5)',
            }}
          />
          {/* Nội dung */}
          <Box
            sx={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '1cqw',
            }}
          >
            <Box component="img" src={`${A}/icon-play.svg`} alt="" sx={{ width: '4.6cqw', height: '4.6cqw' }} />
            <Box
              component="span"
              sx={{
                fontFamily: "'Roboto', sans-serif",
                fontWeight: 600,
                fontSize: '2.5cqw',
                lineHeight: 1,
                color: '#fff',
                whiteSpace: 'nowrap',
              }}
            >
              {t('title_w1_continue', 'Continue')}
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
