# syntax=docker/dockerfile:1
# Build context: starter/docker/mcp/
# Via task:      task mcp:up   (from repo root)
# Manual:        docker build -f gomcp.dockerfile -t go-mcp .
#                docker run -d --name go-mcp -p 8086:8086 -v /path/to/workspace:/workspace:ro go-mcp

ARG PORT=8086

FROM golang:1.26-alpine AS build
WORKDIR /app
ENV GONOSUMDB=*
# Download deps first — cached unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mcp-server .

FROM scratch
ARG PORT
ENV MCP_PORT=${PORT}
COPY --from=build /app/mcp-server /mcp-server
EXPOSE ${PORT}
ENTRYPOINT ["/mcp-server"]
