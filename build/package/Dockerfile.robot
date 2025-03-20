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
RUN go build ./cmd/robot/

EXPOSE 13000

FROM scratch as robot
COPY --from=0 /mango /
CMD ["./robot","-Type=10","-Id=3000","-DockerRun=1"]