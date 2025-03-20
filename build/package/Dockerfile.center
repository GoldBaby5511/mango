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
RUN go build ./cmd/center/

EXPOSE 10050

FROM scratch as center
COPY --from=0 /mango /
CMD ["./center","-Type=2","-Id=50","-DockerRun=1"]