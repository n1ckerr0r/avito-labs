FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trip-service ./cmd/trip-service

# A separate, disposable image applies the Goose migrations in docker compose.
# The service image remains limited to the application binary.
FROM build AS migrate-build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/goose github.com/pressly/goose/v3/cmd/goose

FROM scratch AS migrate
COPY --from=migrate-build /out/goose /goose
COPY migrations /migrations
ENTRYPOINT ["/goose"]

# Keep this stage last so a plain `docker build` produces the service image.
FROM gcr.io/distroless/static-debian12:nonroot AS service
COPY --from=build /out/trip-service /trip-service
EXPOSE 8080
ENTRYPOINT ["/trip-service"]
