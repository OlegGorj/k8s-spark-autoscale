ARG BASE_CONTAINER=ubuntu:bionic-20180526@sha256:c8c275751219dadad8fa56b3ac41ca6cb22219ff117ca98fe82b42f24e1ba64e
FROM $BASE_CONTAINER

USER root

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get -yq dist-upgrade \
 && apt-get install -yq --no-install-recommends \
    git \
    software-properties-common \
    wget \
    bzip2 \
    ca-certificates \
    sudo \
    locales \
    fonts-liberation \
    vim \
  && add-apt-repository ppa:openjdk-r/ppa -y \
  && apt-get install openjdk-8-jdk build-essential -y

RUN apt-get install -y python3-distutils \
    && ln -s /usr/bin/python3 /usr/bin/python \
    && cd ~ && wget https://bootstrap.pypa.io/get-pip.py \
    && python get-pip.py

RUN rm -rf /var/lib/apt/lists/*

RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen

ENV SHELL=/bin/bash \
    LC_ALL=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    LANGUAGE=en_US.UTF-8

ENV SPARK_VERSION 2.2.3
ENV SPARK_PACKAGE spark-${SPARK_VERSION}-bin-hadoop2.7
ENV SPARK_LOC /usr
ENV SPARK_HOME ${SPARK_LOC}/spark
ENV PATH $PATH:${SPARK_HOME}/bin
RUN cd ${SPARK_LOC} && wget http://mirror.dsrg.utoronto.ca/apache/spark/spark-${SPARK_VERSION}/${SPARK_PACKAGE}.tgz \
 && tar -xvzf ${SPARK_PACKAGE}.tgz
RUN ln -s $SPARK_PACKAGE  ${SPARK_LOC}/spark

ARG CACHEBUST=1
# add configs
ADD long_running_job.py $SPARK_HOME/long_running_job.py

WORKDIR $SPARK_HOME



# eof
