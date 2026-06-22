FROM node:20-alpine AS web-build
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.22-alpine AS go-build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-build /src/web/dist ./internal/embed/ui
ARG VERSION=0.1.0-dev
ARG BUILD_COMMIT=unknown
RUN go build -ldflags "-X main.version=${VERSION} -X main.buildCommit=${BUILD_COMMIT}" -o /out/rift ./cmd/rift

FROM alpine:3.19 AS runtime
WORKDIR /app
RUN addgroup -S rift && adduser -S rift -G rift && mkdir -p /app/migrations && chown -R rift:rift /app
COPY --from=go-build /out/rift ./rift
USER rift
EXPOSE 7878
ENTRYPOINT ["./rift"]
CMD ["server"]
