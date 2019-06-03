FROM node:alpine

WORKDIR /proxy
ADD . .

RUN yarn install

EXPOSE 4000
ENTRYPOINT ["node", "index.js"]
