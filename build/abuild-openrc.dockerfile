FROM alpine:3.19
LABEL maintainer="Wang Qi ericwq057@qq.com"
LABEL build_date="2024-02-05"
# ref https://github.com/robertdebock/docker-alpine-openrc/blob/master/Dockerfile

ENV container=docker

ARG ROOT_PWD=inject_from_args
ARG USER_PWD=inject_from_args
ARG SSH_PUB_KEY
ARG HOME=/home/ide

# Enable init.
RUN apk add --update --no-cache sudo openrc openssh-server utmps rsyslog rsyslog-openrc && \
    sed -i 's/^\(tty\d\:\:\)/#\1/g' /etc/inittab && \
    sed -i \
      -e 's/#rc_sys=".*"/rc_sys="docker"/g' \
      -e 's/#rc_env_allow=".*"/rc_env_allow="\*"/g' \
      -e 's/#rc_crashed_stop=.*/rc_crashed_stop=NO/g' \
      -e 's/#rc_crashed_start=.*/rc_crashed_start=YES/g' \
      -e 's/#rc_provide=".*"/rc_provide="loopback net"/g' \
      /etc/rc.conf && \
    rm -f /etc/init.d/hwdrivers \
      /etc/init.d/hwclock \
      /etc/init.d/hwdrivers \
      /etc/init.d/modules \
      /etc/init.d/modules-load \
      /etc/init.d/modloop && \
    sed -i 's/cgroup_add_service /# cgroup_add_service /g' /lib/rc/sh/openrc-run.sh && \
    sed -i 's/VSERVER/DOCKER/Ig' /lib/rc/sh/init.sh

# Create user/group 
# ide/develop
#
RUN addgroup develop && adduser -D -h $HOME -s /bin/ash -G develop ide
# RUN mkdir -p $GOPATH && chown -R ide:develop $GOPATH
RUN echo 'ide ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ide


USER ide:develop
WORKDIR $HOME

# setup ssh for user ide
# setup public key login for normal user
#
RUN mkdir -p $HOME/.ssh \
	&& chmod 0700 $HOME/.ssh \
	&& echo "$SSH_PUB_KEY" > $HOME/.ssh/authorized_keys

USER root

# enable sshd, permit root login, enable port 22, generate ssh key.
#
RUN rc-update add sshd boot \
	&& sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config \
	&& sed -ie 's/#Port 22/Port 22/g' /etc/ssh/sshd_config \
	# && echo '%wheel ALL=(ALL) ALL' > /etc/sudoers.d/wheel \
	&& ssh-keygen -A \
	# && adduser ide wheel \
	&& rm -rf /var/cache/apk/*

# enable rsyslog 
RUN rc-update add rsyslog boot \
   # enable syslog udp 514
   && sed -i \
	-e 's/#module(load="imudp").*/module(load="imudp")/g' \
	-e 's/#input(.*/input(/g' \
	-e 's/#.*type="imudp"/\ttype="imudp"/g' \
	-e 's/#.*port="514"/\tport="514"/g' \
	-e 's/#).*/)/g' \
   /etc/rsyslog.conf

# enable root login, for debug dockerfile purpose.
# set root password
# set ide password
# set root public key login
RUN mkdir -p /root/.ssh \
	&& chmod 0700 /root/.ssh \
	&& echo "root:${ROOT_PWD}" | chpasswd \
	&& echo "ide:${USER_PWD}" | chpasswd \
	&& echo "$SSH_PUB_KEY" > /root/.ssh/authorized_keys

VOLUME ["/sys/fs/cgroup"]

EXPOSE 22
EXPOSE 60000/udp
EXPOSE 60001/udp
EXPOSE 60002/udp
EXPOSE 60003/udp

CMD ["/sbin/init"]
