import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
  })

  test('should show validation errors for empty fields', async ({ page }) => {
    await page.click('button[type="submit"]')
    
    await expect(page.locator('text=Email is required')).toBeVisible()
    await expect(page.locator('text=Password is required')).toBeVisible()
  })

  test('should show error for invalid email format', async ({ page }) => {
    await page.fill('input[name="email"]', 'invalid-email')
    await page.fill('input[name="password"]', 'password123')
    await page.click('button[type="submit"]')
    
    await expect(page.locator('text=Invalid email format')).toBeVisible()
  })

  test('should login successfully with valid credentials', async ({ page }) => {
    // Mock successful login
    await page.route('/api/v1/auth/login', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'mock-jwt-token',
          user: {
            id: 1,
            email: 'admin@alhudhud.com',
            name: 'Admin User',
          },
        }),
      })
    })
    
    await page.fill('input[name="email"]', 'admin@alhudhud.com')
    await page.fill('input[name="password"]', 'securepassword123')
    await page.click('button[type="submit"]')
    
    // Should redirect to dashboard
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 5000 })
  })

  test('should show error for invalid credentials', async ({ page }) => {
    // Mock failed login
    await page.route('/api/v1/auth/login', async route => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Invalid email or password',
        }),
      })
    })
    
    await page.fill('input[name="email"]', 'admin@alhudhud.com')
    await page.fill('input[name="password"]', 'wrongpassword')
    await page.click('button[type="submit"]')
    
    await expect(page.locator('text=Invalid email or password')).toBeVisible()
  })
})

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.route('/api/v1/auth/login', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'mock-jwt-token',
          user: { id: 1, email: 'admin@alhudhud.com', name: 'Admin' },
        }),
      })
    })
    
    await page.goto('/dashboard')
    await page.waitForURL(/\/dashboard/)
  })

  test('should display tenant statistics', async ({ page }) => {
    await page.route('/api/v1/admin/stats/tenants', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          totalTenants: 150,
          activeTenants: 120,
          messagesToday: 50000,
          revenue: 25000.50,
        }),
      })
    })
    
    await page.reload()
    
    await expect(page.locator('text=150')).toBeVisible()
    await expect(page.locator('text=120')).toBeVisible()
  })

  test('should navigate to messaging page', async ({ page }) => {
    await page.click('a[href="/messaging"]')
    await expect(page).toHaveURL(/\/messaging/)
  })
})