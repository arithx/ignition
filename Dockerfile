FROM centos:7
ENV container docker
ARG TARGET
ARG GIMME_ARCH
ARG GIMME_CGO_ENABLED
COPY docker_test.sh /
COPY env_vars.sh /
COPY ./* /ignition/
RUN (cd /lib/systemd/system/sysinit.target.wants/; for i in *; do [ $i == \
systemd-tmpfiles-setup.service ] || rm -f $i; done); \
rm -f /lib/systemd/system/multi-user.target.wants/*;\
rm -f /etc/systemd/system/*.wants/*;\
rm -f /lib/systemd/system/local-fs.target.wants/*; \
rm -f /lib/systemd/system/sockets.target.wants/*udev*; \
rm -f /lib/systemd/system/sockets.target.wants/*initctl*; \
rm -f /lib/systemd/system/basic.target.wants/*;\
rm -f /lib/systemd/system/anaconda.target.wants/*;
VOLUME [ "/sys/fs/cgroup" ]
RUN yum install sudo -y
RUN sudo chmod +x /env_vars.sh
RUN sudo chmod +x /docker_test.sh
CMD ["/usr/sbin/init"]
