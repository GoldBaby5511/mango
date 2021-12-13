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
RUN go build ./servers/center/

RUN mkdir conf
COPY servers/center/conf/center.json conf/

EXPOSE 10001

FROM scratch as center
COPY --from=0 /xlddz /
CMD ["./center","/DockerRun","1"]