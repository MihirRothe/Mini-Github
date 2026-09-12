# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm install

COPY apps/web/ ./
RUN npm run build

# Production runtime stage
FROM caddy:2-alpine

COPY --from=builder /app/dist /srv
COPY apps/web/docker/Caddyfile /etc/caddy/Caddyfile

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:80 || exit 1
