FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-sshd:24.04

ARG GOLANG_VERSION=1.23.6

RUN DEBIAN_FRONTEND=noninteractive apt-get update \
    && apt install -y wget gcc unzip mariadb-client \
    && apt clean

RUN wget https://mirrors.aliyun.com/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && rm -rf /usr/local/go && tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && rm -rf go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile \
    && echo 'export GOPROXY=https://goproxy.cn,direct' >> /etc/profile
