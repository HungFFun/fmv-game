// NotificationStack — toast xếp chồng toàn cục, do SERVER phát (scene.notifications).
// Mỗi toast tự hẹn giờ biến mất → advance liên tục không huỷ timer của toast trước.
import { useEffect } from 'react';

// MUI
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

const NOTIF_ICON = { gallery: '🖼', ending: '🏆', chapter: '📖' };
const NOTIF_TTL_MS = 4500;

export default function NotificationStack({ items, onDismiss }) {
  return (
    <Stack
      aria-live="polite"
      spacing={1.25}
      sx={{ position: 'absolute', top: 60, right: 16, zIndex: 31, pointerEvents: 'none' }}
    >
      {items.map((n) => (
        <NotifToast key={n.id} item={n} onDismiss={onDismiss} />
      ))}
    </Stack>
  );
}

function NotifToast({ item, onDismiss }) {
  useEffect(() => {
    const t = setTimeout(() => onDismiss(item.id), NOTIF_TTL_MS);
    return () => clearTimeout(t);
  }, [item.id, onDismiss]);

  return (
    <Box
      role="status"
      onClick={() => onDismiss(item.id)}
      sx={{
        pointerEvents: 'auto',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        width: 'min(320px, 80vw)',
        bgcolor: 'rgba(26, 22, 34, 0.94)',
        border: '1px solid #3a3148',
        borderLeft: '3px solid',
        borderLeftColor: 'secondary.main',
        borderRadius: 2.5,
        px: 1.75,
        py: 1.25,
        backdropFilter: 'blur(6px)',
        boxShadow: '0 6px 20px rgba(0,0,0,0.4)',
        animation: 'notifIn 0.32s cubic-bezier(0.2, 0.8, 0.2, 1)',
        '@keyframes notifIn': {
          from: { opacity: 0, transform: 'translateX(24px)' },
          to: { opacity: 1, transform: 'translateX(0)' },
        },
        '&:hover': { borderColor: 'primary.main' },
      }}
    >
      <Box component="span" aria-hidden sx={{ fontSize: 22, lineHeight: 1 }}>
        {NOTIF_ICON[item.kind] ?? '🔔'}
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '1px' }}>
        <Typography component="strong" sx={{ fontSize: 13, fontWeight: 600 }}>
          {item.title}
        </Typography>
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{item.body}</Typography>
      </Box>
    </Box>
  );
}
