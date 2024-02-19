FROM golang:1.19 as builder

ADD . /tvhere
RUN cd /tvhere && CGO_ENABLED=0 go build -o tvhere main.go

FROM alpine
RUN apk update && apk add ca-certificates && apk add ffmpeg
COPY --from=builder /tvhere/tvhere /usr/local/bin/
EXPOSE 80
ENTRYPOINT ["tvhere"]