FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /xray

FROM gcr.io/distroless/static-debian12:nonroot as final

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /xray /xray

USER nonroot

EXPOSE 8080

CMD ["/xray"]
