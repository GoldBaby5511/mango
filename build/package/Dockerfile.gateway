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
RUN go build ./cmd/gateway/

EXPOSE 10100 

FROM scratch as gateway
COPY --from=0 /mango /
CMD ["./gateway","-Type=4","-Id=100","-DockerRun=1"]