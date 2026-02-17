# Stage 1: Build
FROM node:20-alpine AS build-stage

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm install

# Copy source and build
COPY . .

# Pass the build argument from docker-compose to the environment
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL

RUN npm run build

# Stage 2: Serve
FROM nginx:stable-alpine

# 1. Clean out the default Nginx files entirely
RUN rm -rf /usr/share/nginx/html/*

# 2. Copy the CONTENTS of your build folder
COPY --from=build-stage /app/dist/client /usr/share/nginx/html

# 3. Copy config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]