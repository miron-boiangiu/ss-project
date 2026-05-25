// defineConfig from vitest/config knows about the `test` block. With the
// yarn `resolutions` pin on `vite`, both vite and vitest see the same Vite
// version so the plugin types line up.
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/setupTests.ts'],
  },
})
