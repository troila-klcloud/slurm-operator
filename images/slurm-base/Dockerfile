FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-base:24.04

COPY sources.list /etc/apt/sources.list

ARG SLURM_REPO="deb [trusted=yes] http://192.168.5.6:80/debs ./"

RUN echo ${SLURM_REPO} > /etc/apt/sources.list.d/slurm.list \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y slurm-smd slurm-smd-client apptainer \
    && apt clean \
    && rm -rf /etc/apt/sources.list.d/slurm.list
