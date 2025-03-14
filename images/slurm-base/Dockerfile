FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-base:24.04

COPY slurm.list /etc/apt/sources.list.d/

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y slurm-smd slurm-smd-client apptainer \
    && apt clean

