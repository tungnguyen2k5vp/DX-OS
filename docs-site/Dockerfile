ARG NODE_BUILDER_IMAGE=node:24.15.0-alpine
ARG NGINX_IMAGE=nginx:1.30.4-alpine

FROM ${NODE_BUILDER_IMAGE} AS build
WORKDIR /workspace/docs-site

COPY docs-site/package.json docs-site/package-lock.json ./
RUN npm ci

COPY docs-site ./
COPY docs ../docs
RUN npm run build

FROM ${NGINX_IMAGE} AS runtime
COPY docs-site/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /workspace/docs-site/build /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=5s --retries=5 --start-period=5s \
  CMD wget -qO- http://127.0.0.1/healthz >/dev/null || exit 1
