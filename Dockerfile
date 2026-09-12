FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/goat-bot .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/goat-bot /goat-bot
ENV STATE_PATH=/data/state.json
VOLUME /data
USER nonroot:nonroot
ENTRYPOINT ["/goat-bot"]
