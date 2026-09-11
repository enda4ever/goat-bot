FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/return-the-goat .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/return-the-goat /return-the-goat
ENV STATE_PATH=/data/state.json
VOLUME /data
USER nonroot:nonroot
ENTRYPOINT ["/return-the-goat"]
