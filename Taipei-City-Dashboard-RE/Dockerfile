# ====== Builder ======
FROM node:22-bookworm-slim AS builder
WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# ====== Runtime (prod, standalone) ======
FROM node:22-bookworm-slim AS prod
WORKDIR /app

ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME=0.0.0.0

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

RUN useradd -r -u 10001 appuser
USER 10001

CMD ["node", "server.js"]

# ====== Dev ======
# Volume-mount + next dev. Build: docker build --target dev
FROM node:22-bookworm-slim AS dev
WORKDIR /app

ENV NODE_ENV=development \
    PORT=3000 \
    HOSTNAME=0.0.0.0

EXPOSE 3000

CMD ["sh", "-c", "npm install && npm run dev -- -H 0.0.0.0 -p 3000"]
