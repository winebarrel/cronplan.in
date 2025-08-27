FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download && go mod verify
COPY main.go /src/
RUN go build -o cronplan.in

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /src/cronplan.in /app/cronplan.in
CMD ["./cronplan.in"]
