FROM golang:alpine
RUN mkdir -p $GOPATH/src/github.com/ahmdrz/music-channel
ADD . $GOPATH/src/github.com/ahmdrz/music-channel
WORKDIR $GOPATH/src/github.com/ahmdrz/music-channel 
RUN go build -o main .
RUN apk add ffmpeg
CMD ["./main"]
