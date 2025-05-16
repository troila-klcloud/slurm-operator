FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-slurm-base:24.04

ARG SLURM_REPO="deb [trusted=yes] http://172.26.142.102:80/debs ./"

RUN echo ${SLURM_REPO} > /etc/apt/sources.list.d/slurm.list \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y slurm-smd-slurmd \
    && apt clean \
    && rm -rf /etc/apt/sources.list.d/slurm.list 

COPY startup.sh /startup.sh

CMD ["/startup.sh"]
