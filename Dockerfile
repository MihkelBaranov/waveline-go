FROM golang

COPY ./ /go/src/github.com/mihkelbaranov/waveline-go

WORKDIR /go/src/github.com/mihkelbaranov/waveline-go

EXPOSE 5000

RUN go get ./

RUN go build main.go

CMD ./main
