FROM fedora:39
LABEL maintainer="ericwq057@qq.com"
LABEL build_date="2024-05-24"

# For installing latest go versions, you would need to add a repository with the latest versions.
# RUN rpm --import https://mirror.go-repo.io/fedora/RPM-GPG-KEY-GO-REPO
# RUN curl -s https://mirror.go-repo.io/fedora/go-repo.repo | tee /etc/yum.repos.d/go-repo.repo

RUN dnf -y install rpm-build rpm-devel rpmlint make python bash coreutils diffutils patch rpmdevtools \
	sudo dnf-plugins-core tree git wget which ripgrep fzf pkgconfig \
	mock rpm-sign pinentry net-tools htop openssh-server sssd \
	&& dnf clean all

# add user/group
# RUN groupadd mock
RUN adduser -g mock packager

# set password for root and sudo
ARG ROOT_PWD=password
ARG USER_PWD=password
RUN echo "root:${ROOT_PWD}" | chpasswd
RUN echo "packager:${USER_PWD}" | chpasswd
# add packager to sudo list
RUN echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
# start automatically at the boot time
RUN systemctl enable sshd.service
RUN sed -i \
	-e 's/#PermitRootLogin.*/PermitRootLogin\ yes/g' \
	-e 's/#PubkeyAuthentication.*/PubkeyAuthentication\ yes/g' \
	-e 's/#Port 22/Port 22/g' \
	/etc/ssh/sshd_config

# https://developer.fedoraproject.org/deployment/rpm/about.html
EXPOSE 22
EXPOSE 8101/udp
EXPOSE 8102/udp
EXPOSE 8103/udp

ENTRYPOINT ["/sbin/init"]
