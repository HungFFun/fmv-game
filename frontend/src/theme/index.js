// MUI theme — map token cũ (styles.css :root) sang palette chuẩn:
// accent→primary, accent-2→secondary, bg→background.default, panel→background.paper,
// text→text.primary, muted→text.secondary.
import { createTheme } from '@mui/material/styles';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#ff5c8a' },
    secondary: { main: '#8a5cff' },
    background: { default: '#0d0b12', paper: '#1a1622' },
    text: { primary: '#f2eef7', secondary: '#8d8499' },
    success: { main: '#6cffa0' },
    error: { main: '#ff6c6c' },
    warning: { main: '#ffd86c' },
    divider: '#322b40',
  },
  shape: { borderRadius: 8 },
  typography: {
    fontFamily: '-apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        'html, body, #root': { height: '100%' },
        body: { overflow: 'hidden' },
      },
    },
    MuiButton: {
      defaultProps: { variant: 'outlined', color: 'inherit' },
      styleOverrides: {
        root: { textTransform: 'none', borderRadius: 8 },
      },
    },
  },
});

export default theme;
