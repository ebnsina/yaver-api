# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static binaries (CGO off) for both the API and the migration runner.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/yaver ./cmd/yaver \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/migrate ./cmd/migrate

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/yaver /app/yaver
COPY --from=build /out/migrate /app/migrate
# cmd/migrate reads ./migrations relative to the working dir.
COPY --from=build /src/migrations /app/migrations
EXPOSE 8080
USER nonroot:nonroot
# Run migrations before boot with:  docker run --entrypoint /app/migrate <image> up
ENTRYPOINT ["/app/yaver"]
