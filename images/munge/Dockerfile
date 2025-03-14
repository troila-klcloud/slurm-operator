FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-base:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y munge libmunge-dev \
    && apt clean \
    && chmod -R 700 /etc/munge /var/log/munge \
    && chmod -R 711 /var/lib/munge \
    && chown -R 0:0 /etc/munge /var/log/munge /var/lib/munge

COPY start_munge.sh /start_munge.sh

CMD ["./start_munge.sh"]
