FROM node:12 as builder
WORKDIR /home/node/app
COPY package.json yarn.lock ./
RUN yarn install

FROM node:12-alpine
WORKDIR /home/node/app

COPY --from=builder /home/node/app ./
COPY . .

EXPOSE 4000
ENTRYPOINT [ "node", "index.js" ]
