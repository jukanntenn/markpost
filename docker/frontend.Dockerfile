FROM node:24-alpine AS builder

RUN npm install -g pnpm

WORKDIR /app

ENV CI=true

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install

COPY . .

RUN pnpm build

RUN cp -r .next/standalone /app/dist && \
    cp -r .next/static /app/dist/.next/static && \
    cp -r public /app/dist/public

FROM node:24-alpine

WORKDIR /app

COPY --from=builder /app/dist ./

ENV HOSTNAME=0.0.0.0
ENV NODE_ENV=production

EXPOSE 3000

CMD ["node", "server.js"]
