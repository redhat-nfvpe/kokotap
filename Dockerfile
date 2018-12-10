FROM golang:latest as builder
ADD . $GOPATH/src/github.com/redhat-nfvpe/kokotap
WORKDIR $GOPATH/src/github.com/redhat-nfvpe/kokotap
RUN go build ./cmd/kokotap_pod && \
    cp $GOPATH/src/github.com/redhat-nfvpe/kokotap/kokotap_pod /bin

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=builder /bin/kokotap_pod /bin
CMD ["/bin/kokotap_pod"]
