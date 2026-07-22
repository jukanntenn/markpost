FROM node:24-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY package.json ./
RUN npm install

COPY index.ts ./
RUN npx tsc --esModuleInterop --target ES2020 --module commonjs --outDir . index.ts

EXPOSE 3002

CMD ["node", "index.js"]
