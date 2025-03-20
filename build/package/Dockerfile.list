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
RUN go build ./cmd/list/

EXPOSE 10080

FROM scratch as list
COPY --from=0 /mango /
CMD ["./list","-Type=6","-Id=80","-DockerRun=1"]