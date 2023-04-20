# module
ARG MODULE_GROUP="internal-share"
ARG MODULE_NAME="synapsor"

FROM registry01.wezhuiyi.com/library/golang:1.17 as builder
ENV GOPROXY https://goproxy.cn,direct
ENV GOSUMDB off
ENV GO111MODULE on

ARG MODULE_NAME

WORKDIR /opt/app/${MODULE_NAME}
COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server && echo "build synapsor success"


# 打包阶段
FROM registry01.wezhuiyi.com/library/centos:7.4

ARG MODULE_GROUP
ARG MODULE_NAME

WORKDIR /data/app/${MODULE_NAME}
ENV LANG=en_US.UTF-8
ENV TZ=Asia/Shanghai

COPY --from=builder /opt/app/${MODULE_NAME}/server bin/

CMD ["/bin/sh", "-c", "/data/app/synapsor/bin/server"]
