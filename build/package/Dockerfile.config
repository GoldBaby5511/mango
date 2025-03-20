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
RUN go build ./cmd/config/

EXPOSE 10060

FROM scratch as config
COPY --from=0 /mango /
CMD ["./config","-Type=3","-Id=60","-DockerRun=1"]