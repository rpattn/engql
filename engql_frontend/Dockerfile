# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm install

# Copy source and build
COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

# This creates the .output folder containing the server
RUN npm run build

# Stage 2: Runtime
FROM node:20-alpine

WORKDIR /app

# Only copy the build output and essential files
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/server.mjs ./server.mjs
COPY --from=builder /app/node_modules ./node_modules

# TanStack Start/Vinxi defaults to port 3000
ENV PORT=3000
ENV NODE_ENV=production

EXPOSE 3000

# Start the generated server
CMD ["node", "run", "start"]
