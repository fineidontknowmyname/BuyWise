FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.23
RUN apk add --no-cache ca-certificates postgresql18-client && adduser -D -H buywise
WORKDIR /app
COPY --from=build /out/ ./
COPY --from=build /src/migrations ./migrations
COPY deploy/runtime-role.sql ./runtime-role.sql
USER buywise
ENV API_ADDR=0.0.0.0:8080
EXPOSE 8080
CMD ["./api"]
