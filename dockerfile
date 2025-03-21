FROM golang:1.24

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o vendor-service ./cmd

EXPOSE 5004

CMD ["./vendor-service"]
