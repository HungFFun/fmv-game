// Màn Album — Figma "Don't Go Home Yet" (node 162:1151). Mở khi bấm Album.
// Trái: hồ sơ nhân vật (ảnh + speech bubble + tên + bảng thông số, dữ liệu demo tĩnh).
// Phải: panel kính chứa lưới ảnh kỷ niệm — DATA-DRIVEN từ GET /api/gallery (mở/khoá).
// Stage 16:9 + CSS container (cqw/cqh) nên mọi thứ scale theo khung 1920×1080.
import { useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';

// Models
import { api } from '@models/game';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import BackButton from '../backButton';

const A = '/album';
const CH = '/chapter'; // tái dùng back.svg + lock-heart.png

const STAGE_SX = {
  position: 'relative',
  width: 'min(100vw, calc(100vh * 16 / 9))',
  aspectRatio: '16 / 9',
  maxHeight: '100vh',
  overflow: 'hidden',
  containerType: 'size',
  background: `url('${A}/bg.png') center / cover no-repeat`,
};

const MOCHIY = "'Mochiy Pop One', 'Roboto', sans-serif";

// Hồ sơ demo của nhân vật (backend chưa có bio → tĩnh theo design).
const PROFILE = {
  name: 'Hana',
  quote: 'Are you really just going to be friends with me?',
  avatars: [`${A}/portrait.png`, `${A}/avatar2.png`, `${A}/avatar3.png`],
};

function MemoryCard({ item, label, tilt }) {
  const unlocked = item.unlocked && item.mediaUrl;
  return (
    <Box
      sx={{
        position: 'relative',
        aspectRatio: '315 / 225',
        background: '#fff',
        border: '0.3cqw solid #fff',
        boxShadow: '0 0.3cqw 0.8cqw rgba(0,0,0,0.25)',
        transform: `rotate(${tilt}deg)`,
        overflow: 'hidden',
      }}
    >
      {unlocked ? (
        <Box
          component="video"
          src={item.mediaUrl}
          muted
          loop
          autoPlay
          playsInline
          sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
        />
      ) : (
        <Box sx={{ position: 'absolute', inset: 0, background: 'linear-gradient(135deg, #c9d2e8 0%, #e7d3df 100%)', filter: 'blur(0.2cqw)' }} />
      )}
      {!unlocked && (
        <>
          <Box
            component="img"
            src={`${CH}/lock-heart.png`}
            alt=""
            aria-hidden
            sx={{ position: 'absolute', top: '38%', left: '50%', transform: 'translate(-50%, -50%)', width: '32%', filter: 'drop-shadow(0 0.2cqw 0.4cqw rgba(0,0,0,0.3))' }}
          />
          <Box
            component="span"
            sx={{
              position: 'absolute',
              bottom: '12%',
              left: '50%',
              transform: 'translateX(-50%)',
              fontFamily: MOCHIY,
              fontSize: '1.05cqw',
              color: '#2b47a7',
              whiteSpace: 'nowrap',
            }}
          >
            {label}
          </Box>
        </>
      )}
    </Box>
  );
}

export default function AlbumScreen({ onClose }) {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);

  useEffect(() => {
    api.gallery().then((r) => setItems(r.items)).catch(() => {});
  }, []);

  const stats = [
    { label: t('album_w1_age', 'Age'), value: t('album_w1_age_v', '24 yo') },
    { label: t('album_w1_birthday', 'Birthday'), value: t('album_w1_birthday_v', 'April 21st') },
    { label: t('album_w1_relationship', 'Relationship'), value: t('album_w1_relationship_v', 'Single') },
    { label: t('album_w1_occupation', 'Occupation'), value: t('album_w1_occupation_v', 'Manager of Company') },
    { label: t('album_w1_physical', 'Physical'), value: t('album_w1_physical_v', '175cm / 53kg') },
    { label: t('album_w1_family', 'Family'), value: t('album_w1_family_v', 'Mom, Dad, Na-na') },
  ];

  return (
    <Box sx={{ position: 'fixed', inset: 0, zIndex: 40, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#000' }}>
      <Box sx={STAGE_SX}>
        {/* Back */}
        <BackButton id="album-back-btn" onClick={onClose} />

        {/* Avatar selector */}
        <Box
          sx={{
            position: 'absolute', left: '50%', top: '3.7cqh', transform: 'translateX(-50%)',
            width: '15.8cqw', height: '8.9cqh', display: 'flex', alignItems: 'center', gap: '1.6cqw', px: '1.5cqw',
            bgcolor: 'rgba(255,255,255,0.5)', border: '0.16cqw dashed #fff', borderRadius: '0.95cqw',
          }}
        >
          {PROFILE.avatars.map((src, i) => (
            <Box
              key={src}
              component="img"
              src={src}
              alt=""
              sx={{
                borderRadius: '50%', objectFit: 'cover', objectPosition: '50% 20%',
                width: i === 0 ? '4.7cqw' : '3.1cqw',
                aspectRatio: '1',
                border: i === 0 ? '0.18cqw solid #fb3797' : 'none',
                boxShadow: '0.2cqw 0.2cqw 0.5cqw rgba(0,0,0,0.25)',
                background: '#fff',
              }}
            />
          ))}
        </Box>

        {/* ALBUM title + divider */}
        <Box
          component="span"
          sx={{ position: 'absolute', left: '87cqw', top: '3.5cqh', fontFamily: "'Roboto', sans-serif", fontWeight: 700, fontSize: '3.1cqw', color: '#fff', whiteSpace: 'nowrap' }}
        >
          {t('album_w1_title', 'ALBUM')}
        </Box>
        <Box aria-hidden sx={{ position: 'absolute', left: '79.3cqw', top: '8.8cqh', width: '20.9cqw', height: '0.56cqh', opacity: 0.5, background: 'linear-gradient(90deg, rgba(255,255,255,0) 4%, #fff 22%)' }} />

        {/* ===== Hồ sơ nhân vật (trái) ===== */}
        {/* Khung ảnh */}
        <Box aria-hidden sx={{ position: 'absolute', left: '4.7cqw', top: '17.96cqh', width: '28.2cqw', height: '36cqh', border: '0.26cqw solid #fff' }} />
        {/* Ảnh chân dung + speech bubble */}
        <Box sx={{ position: 'absolute', left: '5.42cqw', top: '19.2cqh', width: '26.8cqw', height: '33.6cqh', overflow: 'hidden' }}>
          <Box component="img" src={`${A}/portrait.png`} alt="" sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', objectPosition: '60% 25%' }} />
          <Box
            component="p"
            sx={{ position: 'absolute', left: '1cqw', top: '1.5cqh', m: 0, width: '11cqw', fontFamily: MOCHIY, fontSize: '1.36cqw', lineHeight: 1.25, color: '#e22091' }}
          >
            {PROFILE.quote}
          </Box>
        </Box>
        {/* Thẻ tên */}
        <Box
          sx={{
            position: 'absolute', left: '14.65cqw', top: '49.2cqh', bgcolor: '#fff', px: '1.2cqw', py: '0.3cqh',
            boxShadow: '0.5cqw 0.5cqh 0 rgba(42,55,243,0.2)',
          }}
        >
          <Box component="span" sx={{ fontFamily: MOCHIY, fontSize: '2cqw', color: '#e22091', lineHeight: 1.1 }}>
            {PROFILE.name}
          </Box>
        </Box>

        {/* Panel kính mờ sau bảng stats */}
        <Box aria-hidden sx={{ position: 'absolute', left: '5.99cqw', top: '57.8cqh', width: '25.7cqw', height: '38.8cqh', bgcolor: 'rgba(255,255,255,0.7)', filter: 'blur(0.05cqw)' }} />
        {/* Bảng thông số */}
        <Box sx={{ position: 'absolute', left: '7.08cqw', top: '59.6cqh', display: 'flex', flexDirection: 'column', gap: '0.83cqh' }}>
          {stats.map((s) => (
            <Box key={s.label} sx={{ display: 'flex', alignItems: 'center', height: '4.6cqh' }}>
              <Box
                sx={{
                  width: '10.5cqw', height: '100%', display: 'flex', alignItems: 'center', pl: '0.8cqw',
                  borderLeft: '0.14cqw solid #00c8ff',
                  background: 'linear-gradient(90deg, #5d76ba, #69a5d5)',
                  fontFamily: MOCHIY, fontSize: '1.28cqw', color: '#fff',
                }}
              >
                {s.label}
              </Box>
              <Box component="span" sx={{ ml: '1.2cqw', fontFamily: MOCHIY, fontSize: '1.28cqw', color: '#111' }}>
                {s.value}
              </Box>
            </Box>
          ))}
        </Box>

        {/* ===== Lưới ảnh kỷ niệm (phải) ===== */}
        <Box
          sx={{
            position: 'absolute', left: '38.3cqw', top: '17.7cqh', width: '59.06cqw', height: '78.6cqh',
            bgcolor: 'rgba(0,0,0,0.09)', backdropFilter: 'blur(3.5cqw)', borderRadius: '0.95cqw',
            p: '2.2cqw',
            display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '2cqw', alignContent: 'start',
            overflowY: 'auto',
            '&::-webkit-scrollbar': { display: 'none' }, scrollbarWidth: 'none',
          }}
        >
          {items.map((it, i) => (
            <MemoryCard
              key={it.id}
              item={it}
              tilt={[(-2.3), 1.6, -3, -2.3, -0.9, 3.3, 1.5, 1.9, 2][i % 9]}
              label={
                it.unlocked
                  ? it.title
                  : t('album_w1_obtainable', 'Obtainable in Chapter {{n}}', { n: i + 2 })
              }
            />
          ))}
        </Box>
      </Box>
    </Box>
  );
}
