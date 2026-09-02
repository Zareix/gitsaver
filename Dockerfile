FROM golang:1.27.1-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/gitsaver ./cmd/gitsaver


FROM gcr.io/distroless/static-debian12 AS runner

COPY --from=builder /app/gitsaver /app/gitsaver

ENV DESTINATION_PATH=/output
ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD ["/app/gitsaver", "health"]

CMD ["/app/gitsaver"]
