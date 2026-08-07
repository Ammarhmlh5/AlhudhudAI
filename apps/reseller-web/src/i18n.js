import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import translationEN from './locales/en.json';
import translationAR from './locales/ar.json';

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        translation: translationEN
      },
      ar: {
        translation: translationAR
      }
    },
    lng: "ar",
    fallbackLng: "ar",
    interpolation: {
      escapeValue: false
    }
  });

const applyDirection = (lng) => {
  document.documentElement.dir = lng === 'ar' ? 'rtl' : 'ltr';
  document.documentElement.lang = lng;
};

applyDirection(i18n.language);
i18n.on('languageChanged', applyDirection);

export default i18n;
