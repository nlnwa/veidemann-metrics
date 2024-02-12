FROM golang:1.22 as build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/veidemann-metrics .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /go/bin/veidemann-metrics /

ENTRYPOINT ["/veidemann-metrics"]
EXPOSE 9301

