FROM golang:1.13 as build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/veidemann-metrics .

FROM gcr.io/distroless/static-debian10
COPY --from=build /go/bin/veidemann-metrics /

ENTRYPOINT ["/veidemann-metrics"]
EXPOSE 9301

