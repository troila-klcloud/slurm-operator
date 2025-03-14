FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-base:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y sssd ldap-utils \
    && apt clean \
    && mkdir /original-files \
    && mv /var/lib/sss/* /original-files/

COPY startup.sh /startup.sh

VOLUME /var/lib/sss

CMD ["/startup.sh"]
