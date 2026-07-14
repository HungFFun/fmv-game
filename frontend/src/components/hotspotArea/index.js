// HotspotArea — vùng có thể click trên khung video (màn "Tương tác", Figma Frame 57).
// Cửa kính phát sáng nghiêng theo phối cảnh (Vector 7) + marker ❗ (Group 10) ở góc trên-trái.
// Dùng lại cho nhiều video: truyền vị trí (left/top/width/height) + perspective/tilt.
import Box from '@mui/material/Box';

// Components
import HotspotMarker from '@components/hotspotMarker';

const NEON = '#FF0B90'; // stroke cửa theo design
// Glow trắng CHỈ toả ngoài border (box-shadow, không inset) — 2 lớp; px design → cqw/cqh.
const DOOR_GLOW =
  '-0.727cqw -0.517cqh 1.164cqw rgba(255,255,255,0.9), 0.727cqw -0.776cqh 1.164cqw rgba(255,255,255,0.9)';

export default function HotspotArea({
  id,
  label,
  onClick,
  left,
  top,
  width,
  height,
  perspective = '15cqw', // độ sâu phối cảnh → cửa nghiêng nông/sâu
  tilt = -22, // độ nghiêng rotateY (độ)
  showMarker = true,
  sx,
}) {
  const handleKeyDown = (e) => {
    if (e.key === 'Enter' || e.key === ' ') onClick?.();
  };

  return (
    <Box
      id={id}
      role="button"
      tabIndex={0}
      aria-label={label}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      sx={{
        position: 'absolute',
        left,
        top,
        width,
        height,
        cursor: 'pointer',
        perspective,
        '&:hover .hotspot-door': { filter: 'brightness(1.12)' },
        '&:focus-visible': { outline: 'none' },
        '&:focus-visible .hotspot-door': { outline: '0.2cqw solid #fff', outlineOffset: '0.2cqw' },
        ...sx,
      }}
    >
      {/* Cửa kính phát sáng — nghiêng theo phối cảnh (rotateY), viền hồng + glow trắng ngoài (Vector 7) */}
      <Box
        className="hotspot-door"
        sx={{
          position: 'absolute',
          inset: 0,
          borderRadius: '0.6cqw',
          border: `0.873cqw solid ${NEON}`, // stroke-width 16.762px / 1920
          boxShadow: DOOR_GLOW, // glow trắng chỉ ngoài border
          transform: `rotateY(${tilt}deg)`,
          transformOrigin: 'center',
          animation: 'hotspotPulse 1.6s ease-in-out infinite',
          '@keyframes hotspotPulse': {
            '0%, 100%': { opacity: 0.9 },
            '50%': { opacity: 1 },
          },
        }}
      />
      {/* Marker ❗ (Group 10) — tâm ở góc trên-trái, KHÔNG nghiêng */}
      {showMarker && (
        <HotspotMarker
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '5.5cqw',
            height: '5.5cqw',
            transform: 'translate(-50%, -50%)',
          }}
        />
      )}
    </Box>
  );
}
