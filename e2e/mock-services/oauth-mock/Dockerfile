FROM node:24-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY package.json ./
RUN npm install

COPY index.ts ./

EXPOSE 3001

CMD ["node", "--import", "tsx", "index.ts"]
