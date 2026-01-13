FROM golang:1.25.5-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/gitsaver


FROM gcr.io/distroless/static-debian12 AS runner

COPY --from=builder /app/gitsaver /app/gitsaver

ENV DESTINATION_PATH /output
ENV PORT 8080

EXPOSE 8080

CMD ["/app/gitsaver"]
