FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/recipe-app ./example/recipe-app

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=build /out/recipe-app /app/recipe-app
COPY example/recipe-app /app/example/recipe-app
WORKDIR /app/example/recipe-app
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD curl -fsS http://localhost:8080/health || exit 1
CMD ["/app/recipe-app"]
