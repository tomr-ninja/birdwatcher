FROM golang:1.24 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v
RUN go test -v
RUN CGO_ENABLED=0 go build -o /go/bin/app ./cmd/birdwatcher

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/app /
COPY geoip.dat /
EXPOSE 3000
CMD ["/app"]
