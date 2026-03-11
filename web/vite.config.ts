import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import fs from 'fs'

function daemonDiscoveryPlugin() {
  return {
    name: 'daemon-discovery',
    configureServer(server: { middlewares: { use: Function } }) {
      server.middlewares.use('/api/__daemon_lock', (_req: unknown, res: { setHeader: Function; end: Function; statusCode?: number }) => {
        const lockPath = path.resolve(__dirname, '..', '.promptman', '.daemon.lock')
        try {
          const data = fs.readFileSync(lockPath, 'utf-8')
          res.setHeader('Content-Type', 'application/json')
          res.end(data)
        } catch {
          res.statusCode = 404
          res.end(JSON.stringify({ error: 'Daemon not running' }))
        }
      })
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss(), daemonDiscoveryPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
