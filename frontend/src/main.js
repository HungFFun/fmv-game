import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import '@app/i18n'; // khởi tạo i18next singleton (side-effect) trước khi render

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
