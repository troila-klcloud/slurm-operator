FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-slurm-base:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y slurm-smd-slurmdbd \
    && apt clean

CMD ["/usr/sbin/slurmdbd",  "-D"]

