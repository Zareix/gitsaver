FROM golang:1.27.1-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/gitsaver ./cmd/gitsaver


FROM gcr.io/distroless/static-debian12:nonroot AS runner

COPY --from=builder /app/gitsaver /app/gitsaver

ENV DESTINATION_PATH=/output

CMD ["/app/gitsaver"]
