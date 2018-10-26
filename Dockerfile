#docker build --tag s1061123/kokotap -f ./docker/Dockerfile --rm .
#docker push s1061123/kokotap
#FROM golang:latest
FROM golang:alpine
ADD . $GOPATH/src/github.com/redhat-nfvpe/kokotap

WORKDIR $GOPATH/src/github.com/redhat-nfvpe/kokotap

RUN go build ./cmd/kokotap_pod && \
    cp $GOPATH/src/github.com/redhat-nfvpe/kokotap/kokotap_pod /bin
