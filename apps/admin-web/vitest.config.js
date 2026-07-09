import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    include: ['src/**/*.test.{ts,tsx,js,jsx}'],
    exclude: ['src/e2e/**', 'src/api/__tests__/rate_limit*', 'src/api/__tests__/api.test*', 'node_modules', 'dist'],
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
