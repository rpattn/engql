# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

RUN npm run build

# Stage 2: Runtime
FROM node:20-alpine

WORKDIR /app

# Copy entire dist folder
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./

# Install production deps only
RUN npm install --omit=dev

ENV PORT=3000
ENV NODE_ENV=production

EXPOSE 3000

ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0

CMD ["npm", "run", "start"]