FROM centos/systemd
ENV container docker
ARG TARGET
ARG GIMME_ARCH
ARG GIMME_CGO_ENABLED
COPY docker_test.sh /
COPY env_vars.sh /
RUN yum install sudo -y
RUN sudo chmod +x /env_vars.sh
RUN sudo chmod +x /docker_test.sh
