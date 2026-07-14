// HotspotMarker — dấu ❗ của hotspot tương tác (Figma "Group 10", node 170:1030).
// Dùng icon PNG đặt ở public/icon/hotspot-marker-rays.png; tái dùng cho mọi vùng hotspot.
import Box from '@mui/material/Box';

const ICON_SRC = '/icon/hotspot-marker-rays.png';

export default function HotspotMarker({ size = '100%', sx }) {
  return (
    <Box
      component="img"
      src={ICON_SRC}
      alt=""
      aria-hidden
      sx={{
        width: size,
        height: size,
        objectFit: 'contain',
        pointerEvents: 'none',
        userSelect: 'none',
        ...sx,
      }}
    />
  );
}
