FROM golang:1.15.2-alpine

# 环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
	GOPROXY="https://goproxy.cn,direct"

RUN mkdir /mango
WORKDIR /mango

COPY . .
RUN go build ./cmd/room/

EXPOSE 12000

FROM scratch as room
COPY --from=0 /mango /
CMD ["./room","-Type=9","-Id=2000","-DockerRun=1"]