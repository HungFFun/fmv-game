// MapBadge — badge START / FINISH trên bản đồ chapter (Figma "Don't Go Home Yet"
// node 201:2026). Thoi xanh #b3d5ea + 4 góc bo trắng (Rectangle 27) + thoi trang trí
// #dae6f2 ở giữa; pill hồng (badge-banner.svg = Rectangle 31/32) mang chữ label + "CHAPTER"
// nằm ở NỬA TRÊN thoi. Mọi cỡ chữ dùng container-query (cqw) → co theo badge, không theo
// viewport. Định vị badge qua `style` (left/top/width/height, khớp node dữ liệu).

// MUI
import Box from '@mui/material/Box';

const layer = { position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' };

// 4 góc bo trắng ở đỉnh thoi (Rectangle 27) — vẽ trên cùng square rotate45 với thoi nền.
const CORNERS = (
  <Box
    component="svg"
    viewBox="0 0 130.108 130.108"
    preserveAspectRatio="none"
    sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', overflow: 'visible' }}
  >
    <path
      d="M128.693 24.3952V1.41421H106.42M128.693 97.1934V128.693H97.1934M23.6881 128.693H1.41421L1.41421 108.187M1.41421 32.9142L1.41421 1.41421L32.9142 1.41421"
      stroke="#ffffff"
      strokeWidth="2.82843"
      strokeLinecap="round"
      fill="none"
    />
  </Box>
);

export default function MapBadge({ id, label, style }) {
  return (
    <Box id={id} aria-hidden sx={{ position: 'absolute', ...style, containerType: 'size' }}>
      {/* Thoi nền + 4 góc bo (cùng square rotate45) */}
      <Box sx={layer}>
        <Box sx={{ position: 'relative', width: '70.7%', aspectRatio: '1', transform: 'rotate(45deg)', boxShadow: '0 4px 12px rgba(0,0,0,0.2)' }}>
          <Box sx={{ position: 'absolute', inset: 0, background: '#b3d5ea' }} />
          {CORNERS}
        </Box>
      </Box>
      {/* Thoi trang trí ở giữa (viền + nền nhạt) */}
      <Box sx={layer}>
        <Box sx={{ width: '38.5%', aspectRatio: '1', transform: 'rotate(45deg)', border: '1.2cqw solid #dae6f2' }} />
      </Box>
      <Box sx={layer}>
        <Box sx={{ width: '32%', aspectRatio: '1', transform: 'rotate(45deg)', background: '#dae6f2' }} />
      </Box>

      {/* Pill hồng + label — NỬA TRÊN thoi (top 27.4% theo Figma) */}
      <Box
        sx={{
          position: 'absolute',
          top: '27.4%',
          left: '50%',
          transform: 'translateX(-50%)',
          width: '63.8%',
          aspectRatio: '114.85 / 39.4522',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          filter: 'drop-shadow(0 1px 2px rgba(0,0,0,0.2))',
        }}
      >
        <Box component="img" src="/map/badge-banner.svg" alt="" sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%' }} />
        <Box
          component="span"
          sx={{ position: 'relative', color: '#fff', fontFamily: "'Roboto', sans-serif", fontWeight: 800, fontSize: '14cqw', letterSpacing: '0.01em', lineHeight: 1, mb: '3%' }}
        >
          {label}
        </Box>
      </Box>

      {/* "CHAPTER" — ngay dưới pill (top 52.5% theo Figma) */}
      <Box
        component="span"
        sx={{
          position: 'absolute',
          top: '52.5%',
          left: '50%',
          transform: 'translateX(-50%)',
          color: '#203272',
          fontFamily: "'Roboto', sans-serif",
          fontWeight: 700,
          fontSize: '8.6cqw',
          letterSpacing: '0.24em',
          lineHeight: 1,
          whiteSpace: 'nowrap',
        }}
      >
        CHAPTER
      </Box>
    </Box>
  );
}
