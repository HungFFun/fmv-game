// Màn chọn chapter — Figma "Don't Go Home Yet" (node 162:662). DATA-DRIVEN: thẻ chapter lấy từ
// GET /api/chapters (ảnh poster + khoá theo entitlement); chỉ phần trình bày bám theo design.
// Stage giữ tỉ lệ 16:9 và là CSS container (cqw/cqh) nên mọi kích thước/vị trí scale theo khung 1920×1080.
import { useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';

// Models
import { api } from '@models/game';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import BackButton from '../backButton';

const A = '/chapter'; // asset tải từ Figma
const MOCHIY = "'Mochiy Pop One', 'Roboto', sans-serif";

const STAGE_SX = {
  position: 'relative',
  width: 'min(100cqw, calc(100cqh * 16 / 9))',
  aspectRatio: '16 / 9',
  maxHeight: '100cqh',
  overflow: 'hidden',
  containerType: 'size',
  background: `url('${A}/bg.png') center / cover no-repeat`,
};

function ChapterCard({ ch, onOpenChapter, onLocked }) {
  return (
    <Box
      component="button"
      id={`chapter-${ch.idx}-card`}
      type="button"
      title={ch.title}
      aria-label={ch.title}
      onClick={() => (ch.locked ? onLocked(ch.title) : onOpenChapter(ch.id))}
      sx={{
        position: 'relative',
        flex: '0 0 auto',
        width: '23.6cqw',
        p: '0.31cqw', // khung trắng quanh ảnh (nở xuống dưới để chứa tiêu đề khi hover)
        border: 'none',
        background: '#fff',
        cursor: 'pointer',
        zIndex: 1,
        boxShadow: '0 0.4cqw 1cqw rgba(0,0,0,0.3)',
        transition: 'transform 0.15s ease, box-shadow 0.15s ease',
        // Tiêu đề: ẩn (max-height 0) mặc định, hover mở ra bên dưới ảnh — nằm trong card trắng.
        '& .chapter-title': {
          maxHeight: 0,
          opacity: 0,
          mt: 0,
          transition: 'max-height 0.18s ease, opacity 0.18s ease, margin-top 0.18s ease',
        },
        '&:hover': {
          transform: 'translateY(-0.6cqh) scale(1.04)',
          boxShadow: '0 0.8cqw 1.6cqw rgba(0,0,0,0.45)',
          zIndex: 5, // nổi lên trên các thẻ cạnh
        },
        '&:hover .chapter-title, &:focus-visible .chapter-title': { maxHeight: '5em', opacity: 1, mt: '0.6cqh' },
        '&:focus-visible': { outline: '0.2cqw solid #fff', outlineOffset: '0.2cqw' },
      }}
    >
      {/* Vùng ảnh poster (16:9 của thẻ) */}
      <Box sx={{ position: 'relative', width: '100%', aspectRatio: '453 / 414', overflow: 'hidden' }}>
        {ch.posterUrl && (
          <Box
            component="img"
            src={ch.posterUrl}
            alt=""
            sx={{
              position: 'absolute',
              inset: 0,
              width: '100%',
              height: '100%',
              objectFit: 'cover',
              display: 'block',
              filter: ch.locked ? 'blur(0.3cqw)' : 'none',
            }}
          />
        )}
        {/* Lớp sương sáng mờ khi khoá — giữ ảnh tươi màu như design (node 162:721) */}
        {ch.locked && (
          <Box aria-hidden sx={{ position: 'absolute', inset: 0, pointerEvents: 'none', background: 'rgba(255,255,255,0.12)' }} />
        )}
        {/* Viền trắng trong */}
        <Box
          aria-hidden
          sx={{ position: 'absolute', inset: '0.31cqw', border: '0.08cqw solid rgba(255,255,255,0.9)', pointerEvents: 'none' }}
        />
        {/* Khoá trái tim khi chưa mở */}
        {ch.locked && (
          <Box
            component="img"
            src={`${A}/lock-heart.png`}
            alt=""
            aria-hidden
            sx={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              width: '38%',
              height: 'auto',
              filter: 'drop-shadow(0 0.2cqw 0.4cqw rgba(0,0,0,0.35))',
            }}
          />
        )}
      </Box>

      {/* Tai gấp ruy-băng trắng góc trên-phải (đúng rectangle12) + số chapter */}
      <Box
        component="img"
        src={`${A}/corner.svg`}
        alt=""
        aria-hidden
        sx={{ position: 'absolute', top: 0, right: 0, width: '5.8cqw', aspectRatio: '1' }}
      />
      <Box
        component="span"
        sx={{
          position: 'absolute',
          top: '0.6cqh',
          right: '1cqw',
          color: '#fb3797',
          fontFamily: "'Roboto', sans-serif",
          fontWeight: 700,
          fontSize: '2.4cqw',
          lineHeight: 1,
        }}
      >
        {ch.idx}
      </Box>

      {/* Tên chapter — nằm TRONG card trắng, mở ra bên dưới ảnh khi hover (Figma node 191:903) */}
      <Box className="chapter-title" sx={{ overflow: 'hidden' }}>
        <Box
          component="span"
          sx={{
            display: 'block',
            textAlign: 'center',
            fontFamily: MOCHIY,
            color: '#fb3797',
            fontSize: '1.7cqw',
            lineHeight: 1.15,
            px: '0.4cqw',
            wordBreak: 'break-word',
          }}
        >
          {ch.title}
        </Box>
      </Box>
    </Box>
  );
}

export default function ChapterSelectScreen({ onBack, onOpenChapter, onLocked }) {
  const { t } = useTranslation();
  const [chapters, setChapters] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .chapters()
      .then((r) => setChapters(r.chapters))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  return (
    <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#000' }}>
      <Box sx={STAGE_SX}>
        {/* Nút back (chevron «) */}
        <BackButton id="chapter-back-btn" onClick={onBack} />

        {/* Header CHAPTER + đường kẻ + phụ đề */}
        <Box
          component="span"
          sx={{
            position: 'absolute',
            left: '59.5cqw',
            top: '3.5cqh',
            fontFamily: "'Roboto', sans-serif",
            fontWeight: 700,
            fontSize: '8.1cqw',
            lineHeight: 1,
            whiteSpace: 'nowrap',
            background: 'linear-gradient(180deg, rgba(255,255,255,0.8) 53%, rgba(153,153,153,0.4) 111%)',
            WebkitBackgroundClip: 'text',
            backgroundClip: 'text',
            color: 'transparent',
          }}
        >
          {t('chapter_select_w1_title', 'CHAPTER')}
        </Box>
        <Box
          aria-hidden
          sx={{
            position: 'absolute',
            left: '50.3cqw',
            top: '17.2cqh',
            width: '49.9cqw',
            height: '0.9cqh',
            opacity: 0.5,
            background: 'linear-gradient(90deg, rgba(255,255,255,0) 4%, #fff 22%)',
          }}
        />
        <Box
          component="span"
          sx={{
            position: 'absolute',
            left: '59.8cqw',
            top: '20.7cqh',
            opacity: 0.5,
            color: '#fff',
            fontFamily: "'Roboto', sans-serif",
            fontWeight: 700,
            fontSize: '2.46cqw',
            lineHeight: 1,
            whiteSpace: 'nowrap',
          }}
        >
          {t('chapter_select_w1_subtitle', 'Open the chapter!')}
        </Box>

        {/* Hàng thẻ chapter — cuộn ngang, bắt đầu từ trái */}
        <Box
          sx={{
            position: 'absolute',
            left: 0,
            right: 0,
            top: '42cqh',
            display: 'flex',
            alignItems: 'flex-start',
            gap: '4.2cqw',
            pl: '6.25cqw',
            pr: '6.25cqw',
            pt: '2.4cqh', // chừa chỗ cho thẻ nhấc lên khi hover (khỏi bị cắt mép trên)
            pb: '8cqh', // chừa chỗ cho tên chapter hiện khi hover (dưới thẻ)
            overflowX: 'auto',
            overflowY: 'hidden',
            WebkitOverflowScrolling: 'touch',
            '&::-webkit-scrollbar': { display: 'none' },
            scrollbarWidth: 'none',
          }}
        >
          {chapters.map((ch) => (
            <ChapterCard key={ch.id} ch={ch} onOpenChapter={onOpenChapter} onLocked={onLocked} />
          ))}
          {!loading && chapters.length === 0 && (
            <Box component="span" sx={{ color: 'rgba(255,255,255,0.85)', fontSize: '1.8cqw' }}>
              {t('chapter_select_w1_empty', 'Chưa có chapter nào.')}
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
}
