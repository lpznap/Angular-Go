FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server
FROM alpine:3.23.3
RUN apk add --no-cache ca-certificates tzdata curl && addgroup -g 10001 app && adduser -D -u 10001 -G app app && mkdir /data && chown app:app /data
WORKDIR /app
COPY --from=build /server /app/server
COPY docs /app/docs
USER app
EXPOSE 8080
CMD ["/app/server"]
