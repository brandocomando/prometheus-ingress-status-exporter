FROM golang:alpine as builder
RUN apk add --no-cache git
COPY src/* $GOPATH/src/ingress-status-exporter/
WORKDIR $GOPATH/src/ingress-status-exporter
#get dependancies
#you can also use dep
RUN go get -d -v
#build the binary
RUN go build -o /go/bin/ingress-status-exporter

FROM alpine
RUN apk add --no-cache ca-certificates
# Copy our static executable
COPY --from=builder /go/bin/ingress-status-exporter /go/bin/ingress-status-exporter
ENTRYPOINT ["/go/bin/ingress-status-exporter"]
