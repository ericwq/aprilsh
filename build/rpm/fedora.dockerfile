FROM fedora:39
LABEL maintainer="ericwq057@qq.com"

# For installing latest go versions, you would need to add a repository with the latest versions.
# RUN rpm --import https://mirror.go-repo.io/fedora/RPM-GPG-KEY-GO-REPO
# RUN curl -s https://mirror.go-repo.io/fedora/go-repo.repo | tee /etc/yum.repos.d/go-repo.repo

RUN dnf install -y gcc rpm-build rpm-devel rpmlint make python bash coreutils diffutils patch rpmdevtools \
	sudo dnf-plugins-core golang tree git wget which ripgrep fzf pkgconf

# add user/group
RUN groupadd build
RUN adduser -g build packager

# set password for root and sudo
ARG ROOT_PWD=password
ARG USER_PWD=password
RUN echo "root:${ROOT_PWD}" | chpasswd
RUN echo "packager:${USER_PWD}" | chpasswd
# add packager to sudo list
RUN echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager


USER packager:build
WORKDIR /home/packager

CMD ["/bin/bash"]
