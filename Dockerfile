FROM golang:alpine as build

RUN apk update && apk add bash git pkgconfig geoip-dev geoip gcc libc-dev
ADD . /go/src/github.com/m-lab/annotation-service
RUN go get github.com/m-lab/annotation-service
RUN chmod -R a+rx /go/bin/annotation-service

FROM golang:alpine
RUN apk update && apk add geoip
COPY --from=build /go/bin/annotation-service /
WORKDIR /

ENTRYPOINT ["/annotation-service"]
