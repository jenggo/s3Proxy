FROM golang:alpine

WORKDIR /app

COPY go.* ./
RUN go mod download -x

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o s3proxy -trimpath

FROM chainguard/static

COPY --from=0 /app/s3proxy .
COPY views/ views/

EXPOSE 2804

ENTRYPOINT ["./s3proxy"]
