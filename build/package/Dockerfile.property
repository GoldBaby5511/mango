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
RUN go build ./cmd/property/

EXPOSE 10090

FROM scratch as property
COPY --from=0 /mango /
CMD ["./property","-Type=7","-Id=90","-DockerRun=1"]