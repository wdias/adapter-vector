FROM golang:1.13-alpine

WORKDIR /go/src/app
RUN apk update && apk add git
# RUN go get -u github.com/kataras/iris
# Breaking change: https://github.com/kataras/iris/issues/1385#issuecomment-546643215
RUN go get github.com/iris-contrib/cloud-native-go
RUN go get -u github.com/influxdata/influxdb1-client/v2
COPY ./src .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
