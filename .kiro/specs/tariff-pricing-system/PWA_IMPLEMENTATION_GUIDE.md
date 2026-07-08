# دليل تطبيق Progressive Web App (PWA)

## نظرة عامة

هذا الدليل يوفر خطوات تفصيلية لتحويل التطبيق إلى Progressive Web App (PWA) لتوفير تجربة تطبيق أصلي على الويب.

## ما هو PWA؟

Progressive Web App هو تطبيق ويب يستخدم تقنيات حديثة لتوفير تجربة مشابهة للتطبيقات الأصلية:
- يعمل بدون اتصال (Offline)
- قابل للتثبيت على الجهاز
- سريع وموثوق
- آمن (HTTPS)
- قابل للاكتشاف

## المتطلبات الأساسية

### 1. HTTPS
- ✅ جميع الصفحات يجب أن تستخدم HTTPS
- ✅ الشهادة يجب أن تكون صالحة
- ✅ لا توجد محتويات مختلطة (Mixed Content)

### 2. Service Worker
- ✅ ملف Service Worker مسجل
- ✅ يعمل بشكل صحيح
- ✅ يخزن الموارد مؤقتاً

### 3. Web App Manifest
- ✅ ملف manifest.json موجود
- ✅ يحتوي على جميع الحقول المطلوبة
- ✅ الأيقونات بأحجام مختلفة

## خطوات التنفيذ

### الخطوة 1: إنشاء Web App Manifest

#### إنشاء ملف manifest.json

```json
// apps/client-web/public/manifest.json
{
  "name": "نظام التعرفات والتسعير - Hudhud",
  "short_name": "Hudhud Tariff",
  "description": "نظام شامل لإدارة التعرفات والباقات الشهرية",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#020617",
  "theme_color": "#e11d48",
  "orientation": "portrait-primary",
  "scope": "/",
  "lang": "ar",
  "dir": "rtl",
  "icons": [
    {
      "src": "/icons/icon-72x72.png",
      "sizes": "72x72",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-96x96.png",
      "sizes": "96x96",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-128x128.png",
      "sizes": "128x128",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-144x144.png",
      "sizes": "144x144",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-152x152.png",
      "sizes": "152x152",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-384x384.png",
      "sizes": "384x384",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ],
  "screenshots": [
    {
      "src": "/screenshots/dashboard.png",
      "sizes": "1280x720",
      "type": "image/png",
      "label": "لوحة التحكم"
    },
    {
      "src": "/screenshots/packages.png",
      "sizes": "1280x720",
      "type": "image/png",
      "label": "الباقات الشهرية"
    }
  ],
  "categories": ["business", "finance", "productivity"],
  "shortcuts": [
    {
      "name": "إرسال رسالة",
      "short_name": "إرسال",
      "description": "إرسال رسالة جديدة",
      "url": "/send",
      "icons": [
        {
          "src": "/icons/send-96x96.png",
          "sizes": "96x96"
        }
      ]
    },
    {
      "name": "الباقات",
      "short_name": "باقات",
      "description": "عرض الباقات المتاحة",
      "url": "/packages",
      "icons": [
        {
          "src": "/icons/package-96x96.png",
          "sizes": "96x96"
        }
      ]
    }
  ]
}
```

#### إضافة Manifest إلى HTML

```html
<!-- apps/client-web/index.html -->
<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  
  <!-- PWA Manifest -->
  <link rel="manifest" href="/manifest.json">
  
  <!-- Theme Color -->
  <meta name="theme-color" content="#e11d48">
  <meta name="theme-color" media="(prefers-color-scheme: dark)" content="#020617">
  
  <!-- Apple Touch Icon -->
  <link rel="apple-touch-icon" href="/icons/icon-192x192.png">
  
  <!-- Apple Mobile Web App -->
  <meta name="apple-mobile-web-app-capable" content="yes">
  <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
  <meta name="apple-mobile-web-app-title" content="Hudhud Tariff">
  
  <!-- Microsoft Tiles -->
  <meta name="msapplication-TileColor" content="#e11d48">
  <meta name="msapplication-TileImage" content="/icons/icon-144x144.png">
  
  <title>نظام التعرفات والتسعير</title>
</head>
<body>
  <div id="root"></div>
  <script type="module" src="/src/main.jsx"></script>
</body>
</html>
```

### الخطوة 2: إنشاء Service Worker

#### Service Worker الأساسي

```javascript
// apps/client-web/public/sw.js

const CACHE_NAME = 'hudhud-tariff-v1';
const RUNTIME_CACHE = 'hudhud-runtime';

// الموارد المطلوب تخزينها مؤقتاً عند التثبيت
const PRECACHE_URLS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/icons/icon-192x192.png',
  '/icons/icon-512x512.png'
];

// التثبيت - تخزين الموارد الأساسية
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(PRECACHE_URLS))
      .then(() => self.skipWaiting())
  );
});

// التفعيل - تنظيف الذاكرة المؤقتة القديمة
self.addEventListener('activate', (event) => {
  const currentCaches = [CACHE_NAME, RUNTIME_CACHE];
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return cacheNames.filter((cacheName) => !currentCaches.includes(cacheName));
      })
      .then((cachesToDelete) => {
        return Promise.all(
          cachesToDelete.map((cacheToDelete) => {
            return caches.delete(cacheToDelete);
          })
        );
      })
      .then(() => self.clients.claim())
  );
});

// Fetch - استراتيجية التخزين المؤقت
self.addEventListener('fetch', (event) => {
  // تخطي الطلبات غير GET
  if (event.request.method !== 'GET') {
    return;
  }

  // تخطي طلبات API
  if (event.request.url.includes('/api/')) {
    event.respondWith(
      fetch(event.request)
        .catch(() => {
          return new Response(
            JSON.stringify({ error: 'Offline', message: 'لا يوجد اتصال بالإنترنت' }),
            { headers: { 'Content-Type': 'application/json' } }
          );
        })
    );
    return;
  }

  // استراتيجية Cache First للموارد الثابتة
  event.respondWith(
    caches.match(event.request)
      .then((cachedResponse) => {
        if (cachedResponse) {
          return cachedResponse;
        }

        return caches.open(RUNTIME_CACHE)
          .then((cache) => {
            return fetch(event.request)
              .then((response) => {
                // تخزين الاستجابة الناجحة
                if (response.status === 200) {
                  cache.put(event.request, response.clone());
                }
                return response;
              });
          });
      })
      .catch(() => {
        // صفحة Offline
        return caches.match('/offline.html');
      })
  );
});

// Background Sync
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-messages') {
    event.waitUntil(syncMessages());
  }
});

async function syncMessages() {
  // مزامنة الرسائل المعلقة
  const db = await openDB();
  const messages = await db.getAll('pending-messages');
  
  for (const message of messages) {
    try {
      await fetch('/api/v1/messaging/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(message)
      });
      
      await db.delete('pending-messages', message.id);
    } catch (error) {
      console.error('Failed to sync message:', error);
    }
  }
}

// Push Notifications
self.addEventListener('push', (event) => {
  const data = event.data.json();
  
  const options = {
    body: data.body,
    icon: '/icons/icon-192x192.png',
    badge: '/icons/badge-72x72.png',
    vibrate: [200, 100, 200],
    data: {
      url: data.url
    },
    actions: [
      {
        action: 'view',
        title: 'عرض'
      },
      {
        action: 'close',
        title: 'إغلاق'
      }
    ]
  };
  
  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

// Notification Click
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  if (event.action === 'view') {
    event.waitUntil(
      clients.openWindow(event.notification.data.url)
    );
  }
});
```

#### تسجيل Service Worker

```javascript
// apps/client-web/src/registerSW.js

export function registerServiceWorker() {
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker
        .register('/sw.js')
        .then((registration) => {
          console.log('SW registered:', registration);
          
          // التحقق من التحديثات
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            
            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                // يوجد تحديث جديد
                showUpdateNotification();
              }
            });
          });
        })
        .catch((error) => {
          console.error('SW registration failed:', error);
        });
    });
  }
}

function showUpdateNotification() {
  // عرض إشعار للمستخدم بوجود تحديث
  if (confirm('يوجد تحديث جديد. هل تريد تحديث التطبيق؟')) {
    window.location.reload();
  }
}

// طلب إذن الإشعارات
export async function requestNotificationPermission() {
  if ('Notification' in window) {
    const permission = await Notification.requestPermission();
    return permission === 'granted';
  }
  return false;
}

// الاشتراك في Push Notifications
export async function subscribeToPushNotifications() {
  const registration = await navigator.serviceWorker.ready;
  
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(PUBLIC_VAPID_KEY)
  });
  
  // إرسال الاشتراك للخادم
  await fetch('/api/v1/push/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription)
  });
  
  return subscription;
}

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, '+')
    .replace(/_/g, '/');
  
  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  
  return outputArray;
}
```

#### تفعيل Service Worker في التطبيق

```javascript
// apps/client-web/src/main.jsx
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { SocketProvider } from './contexts/SocketContext';
import { registerServiceWorker, requestNotificationPermission } from './registerSW';
import './index.css';
import './rtl.css';
import './i18n/config';
import App from './App.jsx';

// تسجيل Service Worker
registerServiceWorker();

// طلب إذن الإشعارات
requestNotificationPermission();

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <SocketProvider>
          <App />
        </SocketProvider>
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
);
```

### الخطوة 3: إنشاء صفحة Offline

```html
<!-- apps/client-web/public/offline.html -->
<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>لا يوجد اتصال</title>
  <style>
    body {
      font-family: 'Cairo', sans-serif;
      background: #020617;
      color: #fff;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      margin: 0;
      padding: 20px;
      text-align: center;
    }
    
    .container {
      max-width: 400px;
    }
    
    .icon {
      width: 120px;
      height: 120px;
      margin: 0 auto 30px;
      background: #1e293b;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    
    h1 {
      font-size: 24px;
      margin-bottom: 10px;
    }
    
    p {
      color: #94a3b8;
      margin-bottom: 30px;
    }
    
    button {
      background: #e11d48;
      color: white;
      border: none;
      padding: 12px 24px;
      border-radius: 8px;
      font-size: 16px;
      cursor: pointer;
      font-weight: bold;
    }
    
    button:hover {
      background: #be123c;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="icon">
      <svg width="60" height="60" viewBox="0 0 24 24" fill="none" stroke="#e11d48" stroke-width="2">
        <path d="M1 1l6 6m0 0l6-6M7 7l6 6M7 7l-6 6"/>
        <path d="M1 23l6-6m0 0l6 6M7 17l6-6M7 17l-6-6"/>
      </svg>
    </div>
    <h1>لا يوجد اتصال بالإنترنت</h1>
    <p>يبدو أنك غير متصل بالإنترنت. يرجى التحقق من اتصالك والمحاولة مرة أخرى.</p>
    <button onclick="window.location.reload()">إعادة المحاولة</button>
  </div>
</body>
</html>
```

### الخطوة 4: إنشاء الأيقونات

يجب إنشاء أيقونات بأحجام مختلفة:
- 72x72
- 96x96
- 128x128
- 144x144
- 152x152
- 192x192
- 384x384
- 512x512

يمكن استخدام أدوات مثل:
- [PWA Asset Generator](https://github.com/onderceylan/pwa-asset-generator)
- [RealFaviconGenerator](https://realfavicongenerator.net/)

```bash
# استخدام PWA Asset Generator
npx pwa-asset-generator logo.svg ./public/icons
```

### الخطوة 5: إضافة Install Prompt

```javascript
// apps/client-web/src/components/InstallPrompt.jsx
import { useState, useEffect } from 'react';
import { Download, X } from 'lucide-react';

function InstallPrompt() {
  const [deferredPrompt, setDeferredPrompt] = useState(null);
  const [showPrompt, setShowPrompt] = useState(false);

  useEffect(() => {
    const handler = (e) => {
      e.preventDefault();
      setDeferredPrompt(e);
      setShowPrompt(true);
    };

    window.addEventListener('beforeinstallprompt', handler);

    return () => {
      window.removeEventListener('beforeinstallprompt', handler);
    };
  }, []);

  const handleInstall = async () => {
    if (!deferredPrompt) return;

    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;

    if (outcome === 'accepted') {
      console.log('User accepted the install prompt');
    }

    setDeferredPrompt(null);
    setShowPrompt(false);
  };

  const handleDismiss = () => {
    setShowPrompt(false);
    localStorage.setItem('installPromptDismissed', 'true');
  };

  if (!showPrompt || localStorage.getItem('installPromptDismissed')) {
    return null;
  }

  return (
    <div className="fixed bottom-4 left-4 right-4 md:left-auto md:right-4 md:w-96 bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl z-50 animate-in slide-in-from-bottom-4">
      <button
        onClick={handleDismiss}
        className="absolute top-4 right-4 text-slate-400 hover:text-white"
      >
        <X className="w-5 h-5" />
      </button>

      <div className="flex items-start gap-4">
        <div className="w-12 h-12 rounded-xl bg-rose-500/10 flex items-center justify-center flex-shrink-0">
          <Download className="w-6 h-6 text-rose-500" />
        </div>

        <div className="flex-1">
          <h3 className="font-bold text-white mb-2">تثبيت التطبيق</h3>
          <p className="text-sm text-slate-400 mb-4">
            ثبّت التطبيق على جهازك للوصول السريع وتجربة أفضل
          </p>

          <div className="flex gap-3">
            <button
              onClick={handleInstall}
              className="flex-1 bg-rose-600 text-white px-4 py-2 rounded-lg font-bold text-sm hover:bg-rose-700 transition-colors"
            >
              تثبيت
            </button>
            <button
              onClick={handleDismiss}
              className="px-4 py-2 text-slate-400 hover:text-white text-sm font-bold transition-colors"
            >
              لاحقاً
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default InstallPrompt;
```

### الخطوة 6: اختبار PWA

#### استخدام Lighthouse

```bash
# تشغيل Lighthouse
lighthouse https://your-app.com --view
```

#### قائمة التحقق

- [ ] Manifest صحيح ومكتمل
- [ ] Service Worker مسجل ويعمل
- [ ] يعمل بدون اتصال
- [ ] قابل للتثبيت
- [ ] HTTPS مفعّل
- [ ] الأيقونات بأحجام صحيحة
- [ ] Theme color محدد
- [ ] Viewport محدد
- [ ] Lighthouse PWA Score > 90

## المزايا الإضافية

### Background Sync

```javascript
// مزامنة البيانات في الخلفية
async function syncData() {
  if ('sync' in self.registration) {
    try {
      await self.registration.sync.register('sync-messages');
    } catch (error) {
      console.error('Background sync failed:', error);
    }
  }
}
```

### Web Share API

```javascript
// مشاركة المحتوى
async function shareContent() {
  if (navigator.share) {
    try {
      await navigator.share({
        title: 'نظام التعرفات',
        text: 'تحقق من هذا النظام الرائع',
        url: window.location.href
      });
    } catch (error) {
      console.error('Share failed:', error);
    }
  }
}
```

### Add to Home Screen

```javascript
// تتبع التثبيت
window.addEventListener('appinstalled', (event) => {
  console.log('App installed successfully');
  
  // إرسال تحليلات
  gtag('event', 'app_installed');
});
```

## الخلاصة

تطبيق PWA يوفر تجربة مستخدم محسّنة مع إمكانية العمل بدون اتصال والتثبيت على الجهاز. اتبع هذا الدليل لتحويل التطبيق إلى PWA كامل الميزات.

---

**تاريخ الإنشاء:** 2026-03-02  
**الإصدار:** 1.0  
**الحالة:** جاهز للتنفيذ
