FROM golang:1.15.2-alpine

# 环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
	GOPROXY="https://goproxy.cn,direct"

RUN mkdir /xlddz
WORKDIR /xlddz

COPY . .
RUN go build ./servers/gateway/

RUN mkdir conf
COPY servers/gateway/conf/gateway.json conf/

EXPOSE 10102 

FROM scratch as gateway
COPY --from=0 /xlddz /
CMD ["./gateway","/DockerRun","1"]