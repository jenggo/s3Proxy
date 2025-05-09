FROM golang:alpine

RUN apk add ca-certificates

WORKDIR /app

COPY go.* ./
RUN go mod download -x

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o s3proxy -trimpath

FROM scratch

WORKDIR /app

COPY --from=0 /app/s3proxy .
COPY --from=0 /etc/ssl/certs /etc/ssl/certs
COPY views/ views/

EXPOSE 2804

ENTRYPOINT ["./s3proxy"]
