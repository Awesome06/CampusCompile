import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: true, // Required for Docker
    port: 5173,
    proxy: {
      // Intercept any request starting with /api and forward it to the Go backend
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // Ensure Server-Sent Events (SSE) stream smoothly without Vite buffering them
        configure: (proxy, _options) => {
          proxy.on('proxyRes', (proxyRes, req, _res) => {
            if (req.headers.accept === 'text/event-stream') {
              proxyRes.headers['cache-control'] = 'no-cache';
            }
          });
        }
      }
    }
  }
})