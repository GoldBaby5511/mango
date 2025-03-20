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
RUN go build ./cmd/logger/

EXPOSE 20001

FROM scratch as logger
COPY --from=0 /mango /
CMD ["./logger","-Type=1","-Id=1","-DockerRun=1"]