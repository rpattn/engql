import http from 'node:http'
import serverModule from './dist/server/server.js'

// TanStack Start server.js typically default-exports a function
// that already handles requests (req, res)
const handler = serverModule.default ?? serverModule.handler

if (!handler || typeof handler !== 'function') {
  console.error('❌ Could not find a valid SSR handler in dist/server/server.js')
  process.exit(1)
}

const port = process.env.PORT || 3000
const host = process.env.HOST || '0.0.0.0'

const server = http.createServer(handler)

server.listen(port, host, () => {
  console.log(`🚀 SSR Server running at http://${host}:${port}`)
})
