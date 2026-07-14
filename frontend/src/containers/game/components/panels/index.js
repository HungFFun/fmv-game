// AffinityHUD + SaveLoadMenu + GalleryScreen + StoreScreen + Modal (MUI).
// (NotificationStack đã tách ra component toàn cục: @components/notificationStack)
import { useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import CloseIcon from '@mui/icons-material/Close';

// Models
import { api } from '@models/game';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Thanh tiến độ gradient (affinity / timer) dùng chung.
const barGradient = (t) =>
  `linear-gradient(90deg, ${t.palette.primary.main}, ${t.palette.secondary.main})`;

// ===== AffinityHUD =====
export function AffinityHUD({ scene }) {
  return (
    <Stack
      spacing={0.75}
      sx={{
        position: 'absolute',
        left: 16,
        bottom: 16,
        bgcolor: 'rgba(13, 11, 18, 0.7)',
        px: 1.75,
        py: 1.25,
        borderRadius: 2.5,
        backdropFilter: 'blur(6px)',
      }}
    >
      {scene.characters.map((ch) => {
        const v = scene.state.affinity[ch.code] ?? 0;
        return (
          <Box
            key={ch.code}
            title={`${ch.displayName}: ${v}/100`}
            sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 13 }}
          >
            <Box sx={{ width: 70 }}>{ch.displayName}</Box>
            <Box sx={{ width: 120, height: 8, bgcolor: '#2a2336', borderRadius: 1, overflow: 'hidden' }}>
              <Box sx={{ height: '100%', width: `${v}%`, background: barGradient, transition: 'width 0.6s ease' }} />
            </Box>
            <Box sx={{ width: 26, textAlign: 'right', color: 'text.secondary' }}>{v}</Box>
          </Box>
        );
      })}
    </Stack>
  );
}

// ===== SaveLoadMenu =====
export function SaveLoadMenu({ onLoaded, onClose }) {
  const { t } = useTranslation();
  const [saves, setSaves] = useState([]);
  const [msg, setMsg] = useState('');

  const refresh = () => api.saves().then((r) => setSaves(r.saves)).catch(() => {});
  useEffect(() => {
    refresh();
  }, []);

  const slotInfo = (slot) => saves.find((s) => s.slot === slot);

  return (
    <Modal title={t('save_w1_title', 'Save / Load')} onClose={onClose}>
      {[1, 2, 3].map((slot) => {
        const sv = slotInfo(slot);
        return (
          <Box
            key={slot}
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              border: '1px solid #2a2336',
              borderRadius: 2.5,
              px: 1.75,
              py: 1.25,
            }}
          >
            <Box sx={{ display: 'flex', flexDirection: 'column', fontSize: 13, gap: '2px' }}>
              <strong>{t('save_w1_slot', 'Slot {{n}}', { n: slot })}</strong>
              {sv ? (
                <span>
                  {t('save_w1_slot_info', '{{scene}} · chương {{chapter}} · {{time}}', {
                    scene: sv.sceneCode,
                    chapter: sv.chapter,
                    time: sv.updatedAt,
                  })}
                </span>
              ) : (
                <Typography component="span" sx={{ color: 'text.secondary' }}>
                  {t('save_w1_empty', '— trống —')}
                </Typography>
              )}
            </Box>
            <Stack direction="row" spacing={1}>
              <Button
                id={`save-slot-${slot}-save-btn`}
                onClick={() =>
                  api
                    .saveSlot(slot)
                    .then(() => {
                      setMsg(t('save_w1_saved', 'Đã lưu vào slot {{slot}}', { slot }));
                      refresh();
                    })
                    .catch((e) => setMsg(e.message))
                }
              >
                {t('save_w1_save_btn', 'Lưu')}
              </Button>
              <Button
                id={`save-slot-${slot}-load-btn`}
                disabled={!sv}
                onClick={() =>
                  api
                    .loadSlot(slot)
                    .then((s) => {
                      onLoaded(s);
                      onClose();
                    })
                    .catch((e) => setMsg(e.message))
                }
              >
                {t('save_w1_load_btn', 'Tải')}
              </Button>
            </Stack>
          </Box>
        );
      })}
      {msg && (
        <Typography sx={{ color: 'text.secondary' }}>{msg}</Typography>
      )}
    </Modal>
  );
}

// ===== GalleryScreen =====
export function GalleryScreen({ onClose }) {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  useEffect(() => {
    api.gallery().then((r) => setItems(r.items)).catch(() => {});
  }, []);

  return (
    <Modal title={t('gallery_w1_title', 'Gallery')} onClose={onClose}>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 1.5 }}>
        {items.map((it) => (
          <Box key={it.id} sx={{ display: 'flex', flexDirection: 'column', gap: 0.75, fontSize: 13 }}>
            {it.unlocked && it.mediaUrl ? (
              <Box
                component="video"
                src={it.mediaUrl}
                muted
                loop
                autoPlay
                playsInline
                sx={{ width: '100%', aspectRatio: '16/9', objectFit: 'cover', borderRadius: 2 }}
              />
            ) : (
              <Box
                sx={{
                  aspectRatio: '16/9',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  bgcolor: '#14111c',
                  borderRadius: 2,
                  fontSize: 24,
                }}
              >
                🔒
              </Box>
            )}
            <Typography component="span" sx={{ fontSize: 13, color: it.unlocked ? 'text.primary' : 'text.secondary' }}>
              {it.title}
              {it.isBonus && it.unlocked && ' ★'}
            </Typography>
          </Box>
        ))}
        {items.length === 0 && (
          <Typography sx={{ color: 'text.secondary' }}>{t('gallery_w1_empty', 'Chưa có vật phẩm nào.')}</Typography>
        )}
      </Box>
    </Modal>
  );
}

// ===== StoreScreen =====
export function StoreScreen({ highlightChapterId, onPurchased, onClose }) {
  const { t } = useTranslation();
  const [chapters, setChapters] = useState([]);
  const [msg, setMsg] = useState('');

  const refresh = () => api.store().then((r) => setChapters(r.chapters)).catch(() => {});
  useEffect(() => {
    refresh();
  }, []);

  return (
    <Modal title={t('store_w1_title', 'Cửa hàng chương')} onClose={onClose}>
      {chapters.map((ch) => (
        <Box
          key={ch.id}
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            border: '1px solid',
            borderColor: ch.id === highlightChapterId ? 'primary.main' : '#2a2336',
            borderRadius: 2.5,
            px: 1.75,
            py: 1.5,
          }}
        >
          <Box>
            <strong>{ch.title}</strong>
            <Typography component="span" sx={{ color: 'text.secondary' }}>
              {' '}
              · {ch.isFree ? t('store_w1_free', 'Miễn phí') : `$${(ch.priceCents / 100).toFixed(2)}`}
            </Typography>
          </Box>
          {ch.owned ? (
            <Typography component="span" sx={{ color: 'success.main', fontSize: 14 }}>
              {t('store_w1_owned', 'Đã sở hữu ✓')}
            </Typography>
          ) : (
            <Button
              id={`store-chapter-${ch.idx}-buy-btn`}
              onClick={() =>
                api
                  .purchase(ch.id)
                  .then(() => {
                    setMsg(t('store_w1_purchased', 'Mua thành công! (giả lập IAP)'));
                    refresh();
                    onPurchased();
                  })
                  .catch((e) => setMsg(e.message))
              }
            >
              {t('store_w1_buy', 'Mua (giả lập)')}
            </Button>
          )}
        </Box>
      ))}
      {msg && <Typography sx={{ color: 'text.secondary' }}>{msg}</Typography>}
      <Typography sx={{ color: 'text.secondary', fontSize: 12 }}>
        {t(
          'store_w1_dev_note',
          'Dev mode: mua cấp entitlement ngay. Production: entitlement chỉ được cấp qua webhook Stripe/App Store sau khi verify receipt.',
        )}
      </Typography>
    </Modal>
  );
}

// ===== Modal dùng chung (MUI Dialog) =====
function Modal({ title, children, onClose }) {
  return (
    <Dialog
      open
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      slotProps={{
        paper: {
          sx: {
            bgcolor: 'background.paper',
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 3.5,
            p: 2.5,
            gap: 1.5,
            display: 'flex',
            flexDirection: 'column',
            maxHeight: '80vh',
          },
        },
      }}
    >
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6" component="h2">
          {title}
        </Typography>
        <IconButton aria-label="close" size="small" color="inherit" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>
      {children}
    </Dialog>
  );
}
