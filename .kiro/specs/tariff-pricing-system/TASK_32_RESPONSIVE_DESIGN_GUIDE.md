# دليل تطبيق Responsive Design - المهمة 32.1

## نظرة عامة

هذا الدليل يوضح كيفية تطبيق Responsive Design على واجهات المشروع لضمان عملها بشكل ممتاز على جميع الأجهزة.

## الأهداف

- ✅ عمل الواجهات على الهواتف (320px - 767px)
- ✅ عمل الواجهات على الأجهزة اللوحية (768px - 1023px)
- ✅ عمل الواجهات على الشاشات الكبيرة (1024px+)
- ✅ تجربة مستخدم ممتازة على جميع الأحجام

## Breakpoints القياسية

```css
/* Mobile First Approach */

/* Extra Small Devices (Phones) */
@media (max-width: 575px) {
  /* Styles for phones */
}

/* Small Devices (Large Phones) */
@media (min-width: 576px) and (max-width: 767px) {
  /* Styles for large phones */
}

/* Medium Devices (Tablets) */
@media (min-width: 768px) and (max-width: 991px) {
  /* Styles for tablets */
}

/* Large Devices (Desktops) */
@media (min-width: 992px) and (max-width: 1199px) {
  /* Styles for desktops */
}

/* Extra Large Devices (Large Desktops) */
@media (min-width: 1200px) {
  /* Styles for large desktops */
}
```

## التطبيق على واجهة الإدارة

### 1. Dashboard Layout

```jsx
// apps/admin-web/src/pages/Dashboard.tsx

import { useMediaQuery } from '@/hooks/useMediaQuery';

export default function Dashboard() {
  const isMobile = useMediaQuery('(max-width: 767px)');
  const isTablet = useMediaQuery('(min-width: 768px) and (max-width: 1023px)');
  
  return (
    <div className={`dashboard ${isMobile ? 'mobile' : ''} ${isTablet ? 'tablet' : ''}`}>
      {/* Grid Layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {/* Cards */}
      </div>
    </div>
  );
}
```

### 2. Navigation Menu

```jsx
// apps/admin-web/src/components/Navigation.tsx

export default function Navigation() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const isMobile = useMediaQuery('(max-width: 767px)');
  
  return (
    <nav className="navigation">
      {isMobile ? (
        <>
          {/* Hamburger Menu */}
          <button onClick={() => setMobileMenuOpen(!mobileMenuOpen)}>
            <MenuIcon />
          </button>
          
          {/* Mobile Drawer */}
          {mobileMenuOpen && (
            <div className="mobile-menu">
              {/* Menu Items */}
            </div>
          )}
        </>
      ) : (
        <div className="desktop-menu">
          {/* Desktop Menu Items */}
        </div>
      )}
    </nav>
  );
}
```

### 3. Tables

```jsx
// apps/admin-web/src/components/TariffTable.tsx

export default function TariffTable({ tariffs }) {
  const isMobile = useMediaQuery('(max-width: 767px)');
  
  if (isMobile) {
    // Card View for Mobile
    return (
      <div className="tariff-cards">
        {tariffs.map(tariff => (
          <div key={tariff.id} className="tariff-card">
            <div className="card-header">{tariff.service}</div>
            <div className="card-body">
              <div className="field">
                <span className="label">Country:</span>
                <span className="value">{tariff.country}</span>
              </div>
              <div className="field">
                <span className="label">Cost:</span>
                <span className="value">{tariff.cost}</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }
  
  // Table View for Desktop
  return (
    <table className="tariff-table">
      <thead>
        <tr>
          <th>Service</th>
          <th>Country</th>
          <th>Operator</th>
          <th>Cost</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {tariffs.map(tariff => (
          <tr key={tariff.id}>
            <td>{tariff.service}</td>
            <td>{tariff.country}</td>
            <td>{tariff.operator}</td>
            <td>{tariff.cost}</td>
            <td>
              <button>Edit</button>
              <button>Delete</button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

## التطبيق على واجهة العميل

### 1. Dashboard Cards

```jsx
// apps/client-web/src/pages/Dashboard.tsx

export default function Dashboard() {
  return (
    <div className="dashboard">
      {/* Responsive Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <BalanceCard />
        <SubscriptionCard />
        <UsageCard />
      </div>
    </div>
  );
}
```

### 2. Send Message Form

```jsx
// apps/client-web/src/components/SendMessage.tsx

export default function SendMessage() {
  const isMobile = useMediaQuery('(max-width: 767px)');
  
  return (
    <div className={`send-message ${isMobile ? 'mobile' : 'desktop'}`}>
      <form className="message-form">
        {/* Stack vertically on mobile, side-by-side on desktop */}
        <div className="form-layout">
          <div className="recipient-section">
            <label>Recipient</label>
            <input type="tel" />
          </div>
          
          <div className="message-section">
            <label>Message</label>
            <textarea rows={isMobile ? 4 : 6} />
          </div>
          
          <div className="actions-section">
            <button className="btn-primary">Send</button>
            {!isMobile && <button className="btn-secondary">Schedule</button>}
          </div>
        </div>
      </form>
    </div>
  );
}
```

### 3. Package Cards

```jsx
// apps/client-web/src/components/Packages.tsx

export default function Packages() {
  return (
    <div className="packages">
      {/* Responsive Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {packages.map(pkg => (
          <PackageCard key={pkg.id} package={pkg} />
        ))}
      </div>
    </div>
  );
}
```

## CSS Utilities

### 1. Responsive Spacing

```css
/* apps/admin-web/src/styles/responsive.css */

.container {
  padding: 1rem;
}

@media (min-width: 768px) {
  .container {
    padding: 1.5rem;
  }
}

@media (min-width: 1024px) {
  .container {
    padding: 2rem;
  }
}

/* Responsive Margins */
.mb-responsive {
  margin-bottom: 1rem;
}

@media (min-width: 768px) {
  .mb-responsive {
    margin-bottom: 1.5rem;
  }
}

@media (min-width: 1024px) {
  .mb-responsive {
    margin-bottom: 2rem;
  }
}
```

### 2. Responsive Typography

```css
/* Responsive Font Sizes */
.heading-1 {
  font-size: 1.5rem;
}

@media (min-width: 768px) {
  .heading-1 {
    font-size: 2rem;
  }
}

@media (min-width: 1024px) {
  .heading-1 {
    font-size: 2.5rem;
  }
}

.body-text {
  font-size: 0.875rem;
}

@media (min-width: 768px) {
  .body-text {
    font-size: 1rem;
  }
}
```

### 3. Responsive Buttons

```css
.btn {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
}

@media (min-width: 768px) {
  .btn {
    padding: 0.75rem 1.5rem;
    font-size: 1rem;
  }
}

/* Full Width on Mobile */
.btn-responsive {
  width: 100%;
}

@media (min-width: 768px) {
  .btn-responsive {
    width: auto;
  }
}
```

## Custom Hook للـ Media Queries

```typescript
// apps/admin-web/src/hooks/useMediaQuery.ts

import { useState, useEffect } from 'react';

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    const media = window.matchMedia(query);
    
    if (media.matches !== matches) {
      setMatches(media.matches);
    }

    const listener = () => setMatches(media.matches);
    media.addEventListener('change', listener);

    return () => media.removeEventListener('change', listener);
  }, [matches, query]);

  return matches;
}

// Usage Examples
export function useIsMobile() {
  return useMediaQuery('(max-width: 767px)');
}

export function useIsTablet() {
  return useMediaQuery('(min-width: 768px) and (max-width: 1023px)');
}

export function useIsDesktop() {
  return useMediaQuery('(min-width: 1024px)');
}
```

## اختبار Responsive Design

### 1. اختبار يدوي

```bash
# افتح المتصفح وجرب الأحجام التالية:
# - iPhone SE (375x667)
# - iPhone 12 Pro (390x844)
# - iPad (768x1024)
# - iPad Pro (1024x1366)
# - Desktop (1920x1080)
```

### 2. اختبار تلقائي

```javascript
// cypress/e2e/responsive.cy.js

describe('Responsive Design Tests', () => {
  const viewports = [
    { name: 'iPhone SE', width: 375, height: 667 },
    { name: 'iPhone 12 Pro', width: 390, height: 844 },
    { name: 'iPad', width: 768, height: 1024 },
    { name: 'Desktop', width: 1920, height: 1080 }
  ];

  viewports.forEach(viewport => {
    it(`should work on ${viewport.name}`, () => {
      cy.viewport(viewport.width, viewport.height);
      cy.visit('/dashboard');
      
      // Test navigation
      cy.get('[data-testid="navigation"]').should('be.visible');
      
      // Test content
      cy.get('[data-testid="dashboard-content"]').should('be.visible');
      
      // Test buttons
      cy.get('[data-testid="send-message-btn"]').should('be.visible');
    });
  });
});
```

## Best Practices

### 1. Mobile First

```css
/* ✅ Good: Mobile First */
.element {
  width: 100%;
}

@media (min-width: 768px) {
  .element {
    width: 50%;
  }
}

/* ❌ Bad: Desktop First */
.element {
  width: 50%;
}

@media (max-width: 767px) {
  .element {
    width: 100%;
  }
}
```

### 2. Touch-Friendly

```css
/* Larger touch targets on mobile */
.btn-mobile {
  min-height: 44px;
  min-width: 44px;
  padding: 12px 16px;
}

/* Spacing between clickable elements */
.btn-group-mobile > * {
  margin-bottom: 12px;
}
```

### 3. Performance

```jsx
// Lazy load images
<img 
  src={imageSrc} 
  loading="lazy" 
  alt="Description"
/>

// Use srcset for responsive images
<img 
  src="image-small.jpg"
  srcSet="image-small.jpg 480w, image-medium.jpg 768w, image-large.jpg 1200w"
  sizes="(max-width: 768px) 100vw, 50vw"
  alt="Description"
/>
```

## الخلاصة

تطبيق Responsive Design يتطلب:

1. ✅ استخدام Mobile First Approach
2. ✅ تطبيق Breakpoints مناسبة
3. ✅ استخدام Flexbox و Grid
4. ✅ اختبار على أجهزة مختلفة
5. ✅ تحسين الأداء
6. ✅ Touch-Friendly UI

## الخطوات التالية

1. تطبيق Responsive Design على جميع الصفحات
2. اختبار على أجهزة حقيقية
3. إصلاح أي مشاكل
4. توثيق أي استثناءات

---

**الحالة:** جاهز للتطبيق  
**الأولوية:** عالية  
**المدة المتوقعة:** 3-5 أيام
