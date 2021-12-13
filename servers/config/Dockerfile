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
RUN go build ./servers/config/

RUN mkdir conf

COPY servers/config/conf/config.json conf/
COPY servers/config/common-server-center.json .
COPY servers/config/common-server-login.json .
COPY servers/config/xlddz-apollo-application.json .
COPY servers/config/xlddz-gateway-ddz.gateway.json .
COPY servers/config/xlddz-gateway-ddz.gateway-100.json .
COPY servers/config/xlddz-gateway-ddz.gateway-101.json .
COPY servers/config/xlddz-gateway-ddz.gateway-102.json .

EXPOSE 10011

FROM scratch as config
COPY --from=0 /xlddz /
CMD ["./config","/DockerRun","1"]