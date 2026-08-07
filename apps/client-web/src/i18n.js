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
    lng: "ar", // Default language
    fallbackLng: "ar",
    interpolation: {
      escapeValue: false // React cross-site scripting immune
    }
  });

// Update HTML lang and dir when language changes
const applyDirection = (lng) => {
  document.documentElement.lang = lng;
  document.documentElement.dir = lng === 'ar' ? 'rtl' : 'ltr';
};

// Apply on init
applyDirection(i18n.language);

// Apply on language change
i18n.on('languageChanged', applyDirection);

export default i18n;