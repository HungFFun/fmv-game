// i18next singleton — nguồn là TIẾNG VIỆT (sản phẩm VN), `en` là target.
// Side-effect import ở main.js. Backend HTTP nạp /locales/{lng}/common.json theo nhu cầu;
// với useSuspense:false, t() trả về fallback (đối số 2) khi chưa nạp xong → không trắng UI.
import i18n from 'i18next';
import httpBackend from 'i18next-http-backend';
import { initReactI18next } from 'react-i18next';

// Sửa danh sách này khi thêm/bớt ngôn ngữ. Code PHẢI khớp thư mục /public/locales/{code}/.
const supportedLanguages = ['vi', 'en'];

const STORAGE_KEY = 'lang_code';

// 3 tầng phát hiện: localStorage → query ?lng → navigator. Mặc định 'vi'.
const detectLanguage = () => {
  if (typeof window === 'undefined') return 'vi';

  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && supportedLanguages.includes(saved)) return saved;
  } catch {
    /* localStorage không khả dụng */
  }

  try {
    const params = new URLSearchParams(window.location.search);
    const lng = params.get('lng') || params.get('lang');
    if (lng && supportedLanguages.includes(lng)) return lng;
  } catch {
    /* URL không hợp lệ */
  }

  const navLang = navigator.language;
  if (navLang) {
    if (supportedLanguages.includes(navLang)) return navLang;
    const base = navLang.split('-')[0];
    if (supportedLanguages.includes(base)) return base;
  }

  return 'vi';
};

i18n
  .use(httpBackend)
  .use(initReactI18next)
  .init({
    lng: 'en', // ép English cho toàn app (ẩn switcher); bỏ dùng detectLanguage()
    supportedLngs: supportedLanguages,
    fallbackLng: 'en',
    defaultNS: 'common',
    ns: ['common'],
    interpolation: { escapeValue: false }, // React đã chống XSS; chuỗi UI không chứa HTML
    react: { useSuspense: false },
    backend: {
      loadPath: '/locales/{{lng}}/{{ns}}.json',
      allowMultiLoading: false,
    },
    preload: ['en'],
    load: 'currentOnly',
  });

// Lưu lựa chọn ngôn ngữ của người dùng.
i18n.on('languageChanged', (lng) => {
  try {
    localStorage.setItem(STORAGE_KEY, lng);
  } catch {
    /* bỏ qua */
  }
});

export default i18n;
