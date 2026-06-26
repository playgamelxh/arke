FROM node:22-alpine AS builder
WORKDIR /src
COPY package*.json ./

ENV NPM_CONFIG_REGISTRY=https://registry.npmmirror.com

RUN npm ci
COPY . .
ARG VITE_API_BASE=/api
ENV VITE_API_BASE=${VITE_API_BASE}
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/dist /usr/share/nginx/html
EXPOSE 80