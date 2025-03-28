FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y --reinstall ca-certificates \
    && update-ca-certificates \
    && apt clean

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y dumb-init \
    && apt clean


RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt install -y \
       iputils-arping iputils-ping iputils-tracepath \
       git vim make iproute2 sudo \
       apt-transport-https ca-certificates curl gnupg-agent software-properties-common \
       wget python3 gcc python3-dev dumb-init libnss-sss libpam-sss ldap-utils \
       apt-utils tzdata net-tools \
       language-pack-zh-hans language-pack-zh-hant \
    && apt clean


ENTRYPOINT ["dumb-init", "--"]
