FROM golang:1.13-buster as build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /go/bin/app

FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/app /

ENTRYPOINT ["/app"]
EXPOSE 9301

