FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/datasrv ./service/datasrv/cmd

FROM ghcr.io/kongken/go-runtime:main


COPY --from=builder /out/datasrv /app/bin/datasrv

ENTRYPOINT ["/app/bin/datasrv"]
