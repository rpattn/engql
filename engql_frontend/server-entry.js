import { createServer } from 'http'
import handler from './dist/server/server.js'

const port = process.env.PORT || 3000
const host = process.env.HOST || '0.0.0.0'

createServer(handler).listen(port, host, () => {
  console.log(`🚀 Server running at http://${host}:${port}`)
})
