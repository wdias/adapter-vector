FROM golang:1.11.2-alpine

WORKDIR /go/src/app
RUN apk update && apk add git 
RUN go get -u github.com/kataras/iris 
RUN go get -u github.com/influxdata/influxdb/client/v2
COPY ./src .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
