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

# Copy the dist folder exactly as it is
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./

# Set the same production envs
ENV PORT=3000
ENV NODE_ENV=production

EXPOSE 3000

# Run the file directly from the dist/server path
CMD ["node", "dist/server/index.mjs"]