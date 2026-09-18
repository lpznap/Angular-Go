FROM node:24.21.0-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build
FROM nginx:1.28.2-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist/daily/browser /usr/share/nginx/html
EXPOSE 80
