FROM node:20.11.0-alpine3.18

WORKDIR /api

RUN corepack enable yarn

COPY ./package.json /api/package.json
COPY ./.yarn/releases/yarn-4.0.2.cjs /api/.yarn/releases/yarn-4.0.2.cjs
COPY ./yarn.lock /api/yarn.lock
COPY .yarnrc.yml /api/.yarnrc.yml
COPY ./tsconfig.json /api/tsconfig.json

RUN yarn install --immutable

COPY ./src /api/src

RUN yarn build

RUN rm -rf ./src
