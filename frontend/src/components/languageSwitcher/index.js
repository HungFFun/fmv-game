// LanguageSwitcher — nút cờ chuyển ngôn ngữ (dùng chung). Đọc/đổi qua useTranslation.

// MUI
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';

// Hooks
import useTranslation from '@hooks/useTranslation';

export default function LanguageSwitcher({ floating = false }) {
  const { LANGUAGES, currentLanguage, changeLanguage } = useTranslation();
  const current = LANGUAGES.find((l) => currentLanguage?.startsWith(l.code))?.code ?? null;

  return (
    <ToggleButtonGroup
      size="small"
      exclusive
      value={current}
      onChange={(_, code) => code && changeLanguage(code)}
      aria-label="Language"
      sx={{
        ...(floating && { position: 'fixed', top: 12, right: 12, zIndex: 50 }),
        '& .MuiToggleButton-root': {
          px: 0.75,
          py: 0.25,
          fontSize: 16,
          lineHeight: 1,
          border: '1px solid transparent',
          bgcolor: 'rgba(0,0,0,0.35)',
          opacity: 0.55,
          '&:hover': { opacity: 0.85 },
          '&.Mui-selected': { opacity: 1, borderColor: 'primary.main', bgcolor: 'rgba(0,0,0,0.5)' },
        },
      }}
    >
      {LANGUAGES.map((l) => (
        <ToggleButton key={l.code} id={`lang-${l.code}-btn`} value={l.code} title={l.name} aria-label={l.name}>
          {l.flag}
        </ToggleButton>
      ))}
    </ToggleButtonGroup>
  );
}
