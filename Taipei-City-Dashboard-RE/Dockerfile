# ====== Runtime ======
FROM node:22-bookworm-slim AS prod
WORKDIR /app

ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME=0.0.0.0

COPY package.json ./
COPY server.mjs ./
COPY src ./src
COPY public ./public

EXPOSE 3000

RUN useradd -r -u 10001 appuser
USER 10001

CMD ["node", "server.mjs"]

# ====== Dev ======
# Volume-mount + node --watch. Build: docker build --target dev
FROM node:22-bookworm-slim AS dev
WORKDIR /app

ENV NODE_ENV=development \
    PORT=3000 \
    HOSTNAME=0.0.0.0

EXPOSE 3000

CMD ["node", "--watch", "server.mjs"]
