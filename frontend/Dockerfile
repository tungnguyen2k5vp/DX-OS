ARG NODE_BUILDER_IMAGE=node:24.15.0-alpine
ARG NGINX_IMAGE=nginx:1.30.4-alpine

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

FROM ${NGINX_IMAGE}
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/frontend/browser /usr/share/nginx/html
EXPOSE 80
