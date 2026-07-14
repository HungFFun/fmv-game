// VideoNode — 1 node video trên bản đồ chapter (Figma node 191:1889 / 191:1767).
// Định vị tuyệt đối (left/top/width) theo layout map; mặc định là thumbnail + phủ tối
// nhẹ + nút play tròn ở giữa, hover thành card trắng bo góc (đổ bóng, nhấc nhẹ) + hiện
// tiêu đề bên dưới và nổi lên trên các node cạnh. Bấm → onPlay(title).

// MUI
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

const MOCHIY = "'Mochiy Pop One', 'Roboto', sans-serif";


export default function VideoNode({ index, title, posterUrl, style, onPlay }) {
  const { left, top, width } = style ?? {};

  const handleClick = () => onPlay?.(title ?? '');
  const handleKey = (e) => {
    if (e.key === 'Enter' || e.key === ' ') handleClick();
  };

  return (
    <Box
      id={`map-video-${index}`}
      role="button"
      tabIndex={0}
      aria-label={title}
      onClick={handleClick}
      onKeyDown={handleKey}
      sx={{
        position: 'absolute',
        left,
        top,
        width,
        containerType: 'inline-size',
        cursor: 'pointer',
        borderRadius: '3px',
        zIndex: 1,
        transition: 'background-color .18s ease, box-shadow .18s ease, transform .18s ease, padding .18s ease',
        '& .nv-title': {
          maxHeight: 0,
          opacity: 0,
          mt: 0,
          transition: 'max-height .18s ease, opacity .18s ease, margin-top .18s ease',
        },
        '&:hover': {
          bgcolor: '#fff',
          p: '2px',
          boxShadow: '0 2px 5px rgba(0,0,0,0.4)',
          transform: 'translateY(-0.6px)',
          zIndex: 5, // nổi lên trên các node cạnh
        },
        '&:hover .nv-title': { maxHeight: '3.2em', opacity: 1, mt: '4px' },
        '&:focus-visible': { outline: 'none' },
        '&:focus-visible .nv-media': { outline: '0.6px solid #ff2ea6', outlineOffset: '0.4px' },
      }}
    >
      {/* Ảnh + phủ tối + play */}
      <Box
        className="nv-media"
        sx={{
          position: 'relative',
          width: '100%',
          aspectRatio: '16 / 9',
          borderRadius: '2px',
          overflow: 'hidden',
        }}
      >
        {posterUrl ? (
          <Box
            component="img"
            src={posterUrl}
            alt=""
            sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          <Box sx={{ position: 'absolute', inset: 0, bgcolor: '#1a1622' }} />
        )}
        <Box aria-hidden sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.2)' }} />
      </Box>

      {/* Tiêu đề — ẩn mặc định, hiện khi hover */}
      {title && (
        <Box className="nv-title" sx={{ overflow: 'hidden', px: '4px' }}>
          <Typography
            sx={{
              fontFamily: MOCHIY,
              fontWeight: 700,
              color: '#111',
              fontSize: '10px',
              lineHeight: 1.3,
              textAlign: 'center',
              wordBreak: 'break-word',
            }}
          >
            {title}
          </Typography>
        </Box>
      )}
    </Box>
  );
}
