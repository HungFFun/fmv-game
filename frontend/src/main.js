import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import '@app/i18n'; // khởi tạo i18next singleton (side-effect) trước khi render

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Ẩn loading tổng sau khi app đã render (đợi 2 frame để chắc chắn đã vẽ), rồi fade + gỡ.
const loader = document.getElementById('app-loader');
if (loader) {
  requestAnimationFrame(() =>
    requestAnimationFrame(() => {
      loader.classList.add('hidden');
      loader.addEventListener('transitionend', () => loader.remove(), { once: true });
      setTimeout(() => loader.remove(), 800); // fallback nếu transitionend không bắn
    }),
  );
}
