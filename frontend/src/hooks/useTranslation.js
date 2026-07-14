// Wrapper quanh react-i18next: expose t, changeLanguage, currentLanguage, LANGUAGES.
import { useCallback } from 'react';
import { useTranslation as useI18nextTranslation } from 'react-i18next';

// Ngôn ngữ hiển thị trong picker. `code` phải khớp supportedLanguages (src/i18n/index.js).
const LANGUAGES = [
  { name: 'Tiếng Việt', flag: '🇻🇳', code: 'vi' },
  { name: 'English', flag: '🇬🇧', code: 'en' },
];

export const useTranslation = (ns = 'common') => {
  const { t, i18n, ready } = useI18nextTranslation(ns);

  const changeLanguage = useCallback(
    async (langCode) => {
      if (!i18n || !langCode) return;
      try {
        await i18n.changeLanguage(langCode);
      } catch (error) {
        console.error('Failed to change language:', error);
      }
    },
    [i18n],
  );

  return {
    LANGUAGES,
    t,
    i18n,
    isReady: ready,
    currentLanguage: i18n.language,
    changeLanguage,
  };
};

export default useTranslation;
