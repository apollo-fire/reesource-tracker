FROM golang:latest AS go_builder

# Set the working directory and build
WORKDIR /build
RUN mkdir /build/build
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN go mod download
COPY api api
COPY database database
COPY lib lib
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" bash -c ". /root/.bashrc && go build -o ./build/"
COPY ./database/migrations /build/migrations

FROM debian:stable-slim AS reesource_tracker
# Set the working directory and run
WORKDIR /app
RUN apt-get update && apt-get install ca-certificates -y
RUN update-ca-certificates
COPY --from=go_builder /build/build .
COPY client_built ./client
RUN mkdir /app/database && mkdir /app/config
COPY --from=go_builder /build/migrations /app/migrations
# Mount a real config file at /app/config/app.yaml at runtime, e.g.:
#   docker run -v /host/path/app.yaml:/app/config/app.yaml:ro ...
# or override the path via APP_CONFIG_PATH env var.
VOLUME ["/app/config"]
EXPOSE 80
ENTRYPOINT ["/app/reesource-tracker"]
