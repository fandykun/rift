FROM node:24-alpine AS web-build
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
RUN go build -o /out/rift ./cmd/rift

FROM alpine:3.20 AS runtime
WORKDIR /app
RUN addgroup -S rift && adduser -S rift -G rift
COPY --from=go-build /out/rift ./rift
USER rift
EXPOSE 7878
ENTRYPOINT ["./rift"]
CMD ["server"]
