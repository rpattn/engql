# --- Build Stage ---
FROM node:20-alpine AS builder
WORKDIR /app

ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# --- Serve Stage ---
FROM nginx:alpine

# 1. Clean out Nginx defaults
RUN rm -rf /usr/share/nginx/html/*

# 2. Update the Nginx config to point to the "client" subfolder
RUN echo 'server { \
    listen 80; \
    location / { \
        root /usr/share/nginx/html/client; \
        index index.html; \
        try_files $uri $uri/ /index.html; \
    } \
}' > /etc/nginx/conf.d/default.conf

# 3. Copy the ENTIRE dist folder 
# (This keeps the structure you saw in your 'ls' command)
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]