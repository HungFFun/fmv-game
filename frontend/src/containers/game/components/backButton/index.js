// BackButton — nút quay lại (chevron «) dùng chung cho các màn (chapterSelect / chapterMap / album).
// Vị trí góc trên-trái theo design; có thể override qua `sx`.

// MUI
import Box from '@mui/material/Box';

// Hooks
import useTranslation from '@hooks/useTranslation';

export default function BackButton({ id, onClick, sx }) {
  const { t } = useTranslation();
  return (
    <Box
      component="button"
      id={id}
      type="button"
      aria-label={t('common_w1_back', 'Quay lại')}
      onClick={onClick}
      sx={{
        position: 'absolute',
        zIndex: 5,
        left: '1.25cqw',
        top: '2.2cqh',
        width: '5cqw',
        aspectRatio: '1',
        p: '1cqw',
        border: 'none',
        borderRadius: '0.6cqw',
        background: 'transparent',
        cursor: 'pointer',
        transition: 'background 0.12s ease',
        '&:hover': { background: 'rgba(255,255,255,0.18)' },
        '&:focus-visible': { outline: '0.2cqw solid #fff' },
        ...sx,
      }}
    >
      <Box component="img" src="/chapter/back.svg" alt="" sx={{ width: '100%', height: '100%' }} />
    </Box>
  );
}
