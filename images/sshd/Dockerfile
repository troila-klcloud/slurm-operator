FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-slurm-base:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y openssh-server unminimize tmux \
    && systemctl disable ssh.service \
    && apt clean \
    && mkdir -p /run/sshd \
    && yes | unminimize

RUN DEBIAN_FRONTEND=noninteractive pam-auth-update --enable mkhomedir

CMD ["/usr/sbin/sshd", "-D"]
