FROM golang:1.17 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

FROM scratch
COPY --from=builder /app/main /main
ENTRYPOINT [ "/main" ]

