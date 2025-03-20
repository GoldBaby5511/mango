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
RUN go build ./cmd/table/

EXPOSE 11000

FROM scratch as table
COPY --from=0 /mango /
CMD ["./table","-Type=8","-Id=1000","-DockerRun=1"]