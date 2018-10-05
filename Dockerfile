FROM golang:alpine as builder

WORKDIR /go/src/github.com/nlnwa/veidemann-metrics
COPY . .

# Compile the binary statically, so it can be run without libraries.
RUN CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-s -w -static"' .

FROM scratch
COPY --from=builder /go/bin/veidemann-metrics /usr/local/bin/veidemann-metrics

EXPOSE 9301

ENTRYPOINT ["/usr/local/bin/veidemann-metrics"]
