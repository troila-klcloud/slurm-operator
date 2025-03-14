FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-java-tar-pkg:24.04 

RUN tar xvfz jdk-8u431-linux-x64.tar.gz

FROM registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-development:24.04

COPY --from=0 /jdk1.8.0_431 /usr/local/jdk1.8.0_431

ENV JAVA_HOME=/usr/local/jdk1.8.0_431
ENV CLASSPATH=$JAVA_HOME/lib/dt.jar:$JAVA_HOME/lib/tools.jar
ENV PATH=$PATH:$JAVA_HOME/bin

USER root
