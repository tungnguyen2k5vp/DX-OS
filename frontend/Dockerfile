# syntax=docker/dockerfile:1
ARG NODE_BUILDER_IMAGE=node:24.15.0-alpine

FROM ${NODE_BUILDER_IMAGE} AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM dependencies AS test
COPY . .
RUN npm test -- --watch=false

FROM dependencies AS build
COPY . .
RUN npm run build

FROM nginx:1.28.0-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/frontend/browser /usr/share/nginx/html
EXPOSE 80
