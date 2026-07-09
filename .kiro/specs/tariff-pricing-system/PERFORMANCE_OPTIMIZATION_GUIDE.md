# دليل تحسين الأداء - نظام التعرفات والتسعير

## نظرة عامة

هذا الدليل يوفر استراتيجيات شاملة لتحسين أداء التطبيق على الاتصالات البطيئة وضمان تجربة مستخدم سريعة.

## 1. Code Splitting (تقسيم الكود)

### ما هو Code Splitting؟
تقسيم الكود إلى أجزاء صغيرة يتم تحميلها عند الحاجة فقط بدلاً من تحميل كل الكود مرة واحدة.

### التطبيق في React

#### Route-based Code Splitting
```javascript
// apps/client-web/src/App.jsx
import { lazy, Suspense } from 'react';

// تحميل المكونات بشكل كسول
const Dashboard = lazy(() => import('./components/DashboardView'));
const SendMessage = lazy(() => import('./components/SendMessage'));
const Packages = lazy(() => import('./components/Packages'));
const MessageHistory = lazy(() => import('./components/MessageHistory'));

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/send" element={<SendMessage />} />
        <Route path="/packages" element={<Packages />} />
        <Route path="/history" element={<MessageHistory />} />
      </Routes>
    </Suspense>
  );
}
```

#### Component-based Code Splitting
```javascript
// تحميل المكونات الثقيلة بشكل كسول
const HeavyChart = lazy(() => import('./components/HeavyChart'));
const DataTable = lazy(() => import('./components/DataTable'));

function Dashboard() {
  return (
    <div>
      <Suspense fallback={<div>Loading chart...</div>}>
        <HeavyChart />
      </Suspense>
      
      <Suspense fallback={<div>Loading table...</div>}>
        <DataTable />
      </Suspense>
    </div>
  );
}
```

### تكوين Vite للـ Code Splitting

```javascript
// vite.config.js
export default {
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // تقسيم المكتبات الكبيرة
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'chart-vendor': ['recharts'],
          'ui-vendor': ['lucide-react'],
          'i18n-vendor': ['i18next', 'react-i18next']
        }
      }
    },
    chunkSizeWarningLimit: 1000
  }
};
```

## 2. Lazy Loading (التحميل الكسول)

### تحميل الصور بشكل كسول

```javascript
// مكون LazyImage
import { useState, useEffect, useRef } from 'react';

function LazyImage({ src, alt, placeholder }) {
  const [imageSrc, setImageSrc] = useState(placeholder);
  const [imageRef, setImageRef] = useState();

  useEffect(() => {
    let observer;
    
    if (imageRef && imageSrc === placeholder) {
      observer = new IntersectionObserver(
        entries => {
          entries.forEach(entry => {
            if (entry.isIntersecting) {
              setImageSrc(src);
              observer.unobserve(imageRef);
            }
          });
        },
        { rootMargin: '50px' }
      );
      
      observer.observe(imageRef);
    }
    
    return () => {
      if (observer && imageRef) {
        observer.unobserve(imageRef);
      }
    };
  }, [imageRef, imageSrc, placeholder, src]);

  return (
    <img
      ref={setImageRef}
      src={imageSrc}
      alt={alt}
      loading="lazy"
    />
  );
}
```

### تحميل البيانات بشكل كسول

```javascript
// استخدام Intersection Observer لتحميل البيانات
function InfiniteScroll({ loadMore, hasMore }) {
  const observerTarget = useRef(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        if (entries[0].isIntersecting && hasMore) {
          loadMore();
        }
      },
      { threshold: 1 }
    );

    if (observerTarget.current) {
      observer.observe(observerTarget.current);
    }

    return () => {
      if (observerTarget.current) {
        observer.unobserve(observerTarget.current);
      }
    };
  }, [loadMore, hasMore]);

  return <div ref={observerTarget} />;
}
```

## 3. تحسين الصور

### استخدام صيغ حديثة

```html
<!-- استخدام WebP مع Fallback -->
<picture>
  <source srcset="image.webp" type="image/webp">
  <source srcset="image.jpg" type="image/jpeg">
  <img src="image.jpg" alt="Description">
</picture>
```

### ضغط الصور

```bash
# استخدام أدوات ضغط الصور
npm install -D imagemin imagemin-webp imagemin-mozjpeg imagemin-pngquant

# سكريبت ضغط الصور
node scripts/optimize-images.js
```

```javascript
// scripts/optimize-images.js
import imagemin from 'imagemin';
import imageminWebp from 'imagemin-webp';
import imageminMozjpeg from 'imagemin-mozjpeg';
import imageminPngquant from 'imagemin-pngquant';

await imagemin(['public/images/*.{jpg,png}'], {
  destination: 'public/images/optimized',
  plugins: [
    imageminMozjpeg({ quality: 80 }),
    imageminPngquant({ quality: [0.6, 0.8] }),
    imageminWebp({ quality: 80 })
  ]
});
```

### Responsive Images

```html
<!-- أحجام مختلفة للشاشات المختلفة -->
<img
  srcset="
    image-320w.jpg 320w,
    image-640w.jpg 640w,
    image-1280w.jpg 1280w
  "
  sizes="(max-width: 320px) 280px,
         (max-width: 640px) 600px,
         1200px"
  src="image-640w.jpg"
  alt="Description"
>
```

## 4. تحسين الخطوط

### استخدام font-display

```css
/* apps/client-web/src/index.css */
@font-face {
  font-family: 'Cairo';
  src: url('/fonts/Cairo-Regular.woff2') format('woff2');
  font-display: swap; /* عرض خط احتياطي فوراً */
  font-weight: 400;
  font-style: normal;
}
```

### تحميل الخطوط بشكل مسبق

```html
<!-- index.html -->
<link rel="preload" href="/fonts/Cairo-Regular.woff2" as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/fonts/Cairo-Bold.woff2" as="font" type="font/woff2" crossorigin>
```

### استخدام خطوط النظام كـ Fallback

```css
body {
  font-family: 'Cairo', -apple-system, BlinkMacSystemFont, 'Segoe UI', 
               'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 
               'Droid Sans', 'Helvetica Neue', sans-serif;
}
```

## 5. تحسين JavaScript

### Tree Shaking

```javascript
// ✅ استيراد فقط ما تحتاجه
import { useState, useEffect } from 'react';
import { Send, User } from 'lucide-react';

// ❌ تجنب استيراد كل شيء
import * as React from 'react';
import * as Icons from 'lucide-react';
```

### تصغير الكود

```javascript
// vite.config.js
export default {
  build: {
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true, // إزالة console.log في الإنتاج
        drop_debugger: true
      }
    }
  }
};
```

### استخدام Web Workers للعمليات الثقيلة

```javascript
// workers/dataProcessor.js
self.addEventListener('message', (e) => {
  const { data } = e.data;
  
  // معالجة البيانات الثقيلة
  const processed = heavyProcessing(data);
  
  self.postMessage({ processed });
});

// استخدام Worker
const worker = new Worker('/workers/dataProcessor.js');

worker.postMessage({ data: largeDataset });

worker.addEventListener('message', (e) => {
  const { processed } = e.data;
  // استخدام البيانات المعالجة
});
```

## 6. تحسين CSS

### إزالة CSS غير المستخدم

```bash
# استخدام PurgeCSS
npm install -D @fullhuman/postcss-purgecss

# postcss.config.js
module.exports = {
  plugins: [
    require('@fullhuman/postcss-purgecss')({
      content: ['./src/**/*.{js,jsx,ts,tsx}'],
      defaultExtractor: content => content.match(/[\w-/:]+(?<!:)/g) || []
    })
  ]
};
```

### استخدام CSS Modules

```javascript
// Component.module.css
.button {
  padding: 10px 20px;
  background: #e11d48;
}

// Component.jsx
import styles from './Component.module.css';

function Component() {
  return <button className={styles.button}>Click</button>;
}
```

### تصغير CSS

```javascript
// vite.config.js
export default {
  build: {
    cssMinify: true
  }
};
```

## 7. Caching (التخزين المؤقت)

### Service Worker للتخزين المؤقت

```javascript
// public/sw.js
const CACHE_NAME = 'tariff-system-v1';
const urlsToCache = [
  '/',
  '/index.css',
  '/index.js',
  '/fonts/Cairo-Regular.woff2'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(urlsToCache))
  );
});

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request)
      .then((response) => response || fetch(event.request))
  );
});
```

### HTTP Caching Headers

```nginx
# nginx.conf
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
  expires 1y;
  add_header Cache-Control "public, immutable";
}

location ~* \.(html)$ {
  expires -1;
  add_header Cache-Control "no-cache, no-store, must-revalidate";
}
```

## 8. تحسين الشبكة

### استخدام CDN

```html
<!-- استخدام CDN للمكتبات الشائعة -->
<link rel="preconnect" href="https://cdn.jsdelivr.net">
<link rel="dns-prefetch" href="https://cdn.jsdelivr.net">
```

### HTTP/2 Server Push

```nginx
# nginx.conf
location / {
  http2_push /css/main.css;
  http2_push /js/main.js;
}
```

### تقليل حجم الطلبات

```javascript
// استخدام Compression
import compression from 'compression';

app.use(compression({
  level: 6,
  threshold: 1024,
  filter: (req, res) => {
    if (req.headers['x-no-compression']) {
      return false;
    }
    return compression.filter(req, res);
  }
}));
```

## 9. تحسين React

### استخدام React.memo

```javascript
// تجنب إعادة الرسم غير الضرورية
const ExpensiveComponent = React.memo(({ data }) => {
  return <div>{/* render expensive content */}</div>;
}, (prevProps, nextProps) => {
  // إرجاع true إذا كانت الخصائص متساوية (لا حاجة لإعادة الرسم)
  return prevProps.data === nextProps.data;
});
```

### استخدام useMemo و useCallback

```javascript
function Component({ items }) {
  // حفظ النتيجة المحسوبة
  const expensiveValue = useMemo(() => {
    return items.reduce((acc, item) => acc + item.value, 0);
  }, [items]);

  // حفظ الدالة
  const handleClick = useCallback(() => {
    console.log('Clicked');
  }, []);

  return <div onClick={handleClick}>{expensiveValue}</div>;
}
```

### Virtualization للقوائم الطويلة

```bash
npm install react-window
```

```javascript
import { FixedSizeList } from 'react-window';

function LongList({ items }) {
  const Row = ({ index, style }) => (
    <div style={style}>
      {items[index].name}
    </div>
  );

  return (
    <FixedSizeList
      height={600}
      itemCount={items.length}
      itemSize={50}
      width="100%"
    >
      {Row}
    </FixedSizeList>
  );
}
```

## 10. قياس الأداء

### استخدام Lighthouse

```bash
# تشغيل Lighthouse من سطر الأوامر
npm install -g lighthouse

lighthouse https://your-app.com --view
```

### استخدام Web Vitals

```bash
npm install web-vitals
```

```javascript
// src/reportWebVitals.js
import { getCLS, getFID, getFCP, getLCP, getTTFB } from 'web-vitals';

function sendToAnalytics(metric) {
  console.log(metric);
  // إرسال إلى خدمة التحليلات
}

getCLS(sendToAnalytics);
getFID(sendToAnalytics);
getFCP(sendToAnalytics);
getLCP(sendToAnalytics);
getTTFB(sendToAnalytics);
```

### استخدام Performance API

```javascript
// قياس وقت تحميل المكون
function Component() {
  useEffect(() => {
    const start = performance.now();
    
    // عمليات ثقيلة
    
    const end = performance.now();
    console.log(`Component loaded in ${end - start}ms`);
  }, []);
}
```

## معايير الأداء المستهدفة

### Core Web Vitals
- **LCP (Largest Contentful Paint)**: < 2.5s
- **FID (First Input Delay)**: < 100ms
- **CLS (Cumulative Layout Shift)**: < 0.1

### أوقات التحميل
- **First Contentful Paint**: < 1.8s
- **Time to Interactive**: < 3.8s
- **Speed Index**: < 3.4s

### حجم الملفات
- **JavaScript Bundle**: < 200KB (gzipped)
- **CSS Bundle**: < 50KB (gzipped)
- **Total Page Size**: < 1MB

## خطة التنفيذ

### المرحلة 1: التحليل
1. قياس الأداء الحالي باستخدام Lighthouse
2. تحديد الاختناقات
3. تحديد الأولويات

### المرحلة 2: التحسينات السريعة
1. تطبيق Code Splitting
2. تحسين الصور
3. تطبيق Lazy Loading

### المرحلة 3: التحسينات المتقدمة
1. تطبيق Service Worker
2. تحسين CSS و JavaScript
3. تطبيق Virtualization

### المرحلة 4: القياس والتحسين
1. قياس الأداء بعد التحسينات
2. مقارنة مع المعايير المستهدفة
3. تحسينات إضافية حسب الحاجة

## الخلاصة

تحسين الأداء عملية مستمرة. اتبع هذا الدليل لضمان تجربة مستخدم سريعة على جميع الاتصالات.

---

**تاريخ الإنشاء:** 2026-03-02  
**الإصدار:** 1.0  
**الحالة:** جاهز للتنفيذ
