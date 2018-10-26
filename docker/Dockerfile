#docker build --tag s1061123/kokotap -f ./docker/Dockerfile --rm .
#docker push s1061123/kokotap
#FROM golang:latest
FROM golang:alpine
ADD . $GOPATH/src/kokotap

RUN cd $GOPATH/src/kokotap && \
    go build ./cmd/kokotap_pod && \
    cp $GOPATH/src/kokotap/kokotap_pod /bin
