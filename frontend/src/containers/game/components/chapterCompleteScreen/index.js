// ChapterCompleteScreen — màn "Chapter Complete" khi video ending kết thúc.
// Figma "Don't Go Home Yet" node 204:2566: lớp tim phát sáng (soft-light) làm nền.
// Hiện SAU khi clip cuối phát xong (GameShell điều khiển), kèm rank + tên ending + hành động.
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

import useTranslation from '@hooks/useTranslation';

const RANK_COLOR = { good: '#6cffa0', bad: '#ff6c6c', true: '#ffd86c', normal: '#c9d6ff' };

const STAGE_SX = {
  position: 'absolute',
  inset: 0,
  overflow: 'hidden',
  containerType: 'size',
  // nền: gradient xanh→hồng của game + lớp tim phát sáng (soft-light) như node 204:2566
  background: 'linear-gradient(120deg, #2f7fe6 0%, #7b6bd6 46%, #ff56b9 100%)',
};

export default function ChapterCompleteScreen({ ending, onRestart, onGallery }) {
  const { t } = useTranslation();
  const rank = ending?.rank;

  return (
    <Box sx={STAGE_SX}>
      {/* Lớp tim phát sáng (node 204:2566): soft-light, mờ 45% */}
      <Box
        aria-hidden
        sx={{
          position: 'absolute',
          inset: 0,
          backgroundImage: "url('/chapter/complete-bg.jpg')",
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          mixBlendMode: 'soft-light',
          opacity: 0.9,
          pointerEvents: 'none',
        }}
      />
      {/* Scrim tối nhẹ để chữ trắng đọc rõ mà vẫn thấy nền tim */}
      <Box aria-hidden sx={{ position: 'absolute', inset: 0, background: 'radial-gradient(120% 90% at 50% 42%, rgba(10,6,20,0.10) 0%, rgba(10,6,20,0.5) 100%)', pointerEvents: 'none' }} />

      <Stack
        alignItems="center"
        justifyContent="center"
        sx={{ position: 'absolute', inset: 0, textAlign: 'center', px: '6cqw' }}
      >
        {/* Trái tim nhấn phía trên tiêu đề */}
        <Box aria-hidden sx={{ fontSize: '9cqw', lineHeight: 1, mb: '1cqh', filter: 'drop-shadow(0 0.6cqh 1.4cqh rgba(0,0,0,0.35))' }}>💗</Box>

        {rank && (
          <Typography sx={{ fontFamily: "'Roboto', sans-serif", letterSpacing: '0.5cqw', fontWeight: 700, fontSize: '1.7cqw', color: RANK_COLOR[rank] ?? 'rgba(255,255,255,0.85)', mb: '1cqh' }}>
            {rank.toUpperCase()} {t('ending_w1_suffix', 'ENDING')}
          </Typography>
        )}

        <Typography
          component="h1"
          sx={{
            fontFamily: "'Roboto', sans-serif", fontWeight: 900, fontSize: '7cqw', lineHeight: 1.02,
            letterSpacing: '0.3cqw', textTransform: 'uppercase', textWrap: 'balance',
            background: 'linear-gradient(180deg, #ffffff 45%, rgba(255,255,255,0.55) 115%)',
            WebkitBackgroundClip: 'text', backgroundClip: 'text', color: 'transparent',
            textShadow: '0 0.4cqh 2cqh rgba(0,0,0,0.25)',
          }}
        >
          {t('chapter_complete_w1_title', 'Chapter Complete')}
        </Typography>

        {ending?.title && (
          <Typography sx={{ mt: '1.2cqh', fontSize: '2.4cqw', color: 'rgba(255,255,255,0.92)', fontWeight: 600, textShadow: '0 2px 8px rgba(0,0,0,0.4)' }}>
            {ending.title}
          </Typography>
        )}

        <Stack direction="column" alignItems="center" spacing={1.25} sx={{ mt: '3.4cqh' }}>
          <Button
            id="complete-restart-btn"
            variant="contained"
            onClick={onRestart}
            sx={(th) => ({
              background: `linear-gradient(135deg, ${th.palette.primary.main}, ${th.palette.secondary.main})`,
              color: '#fff', fontWeight: 700, px: '3.5cqw', py: '1.4cqh', fontSize: '1.8cqw', borderRadius: '1.2cqw',
              boxShadow: '0 0.6cqh 2cqh rgba(0,0,0,0.35)',
            })}
          >
            {t('ending_w1_restart', 'Chơi lại từ đầu')}
          </Button>
          <Button id="complete-gallery-btn" onClick={onGallery} sx={{ color: '#fff', fontSize: '1.6cqw', textDecoration: 'underline', textUnderlineOffset: '3px' }}>
            {t('ending_w1_view_gallery', 'Xem Gallery')}
          </Button>
        </Stack>
      </Stack>
    </Box>
  );
}
