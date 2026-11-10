# One image holds all three programs: the API server, the scanner and the
# blog writer. Build from the repo root:
#   docker build -f deploy/minipc/Dockerfile .
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/api/mcp-go-fixed ./cmd/api/mcp-go-fixed
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -o /out/scan ./cmd/scan \
 && CGO_ENABLED=0 go build -o /out/blog ./cmd/blog

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/ /app/
ENV PORT=8080
EXPOSE 8080
CMD ["/app/api"]
