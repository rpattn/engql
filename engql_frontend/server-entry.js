// server-entry.js
import { createServer } from '@tanstack/start-server-core'
import { handler } from './dist/server/server.js' // named export

const port = process.env.PORT || 3010
const host = process.env.HOST || '0.0.0.0'

const server = createServer(handler)

server.listen(port, host, () => {
  console.log(`🚀 SSR Server running at http://${host}:${port}`)
})

