import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { fileURLToPath, URL } from 'url'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@alhudhud/shared/api': fileURLToPath(new URL('../libs/shared/src/api/axios.ts', import.meta.url)),
      '@alhudhud/shared': fileURLToPath(new URL('../libs/shared/src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
  },
})
