FROM fedora:39
LABEL maintainer="ericwq057@qq.com"
#LABEL build_date="2024-05-24"

# set password for root and sudo
ARG ROOT_PWD=password
ARG USER_PWD=password

# For installing latest go versions, you would need to add a repository with the latest versions.
# RUN rpm --import https://mirror.go-repo.io/fedora/RPM-GPG-KEY-GO-REPO
# RUN curl -s https://mirror.go-repo.io/fedora/go-repo.repo | tee /etc/yum.repos.d/go-repo.repo

# add sssd to avoid warning:
# PAM unable to dlopen(/usr/lib64/security/pam_sss.so): /usr/lib64/security/pam_sss.so: cannot open shared object file: No such file or directory
# PAM adding faulty module: /usr/lib64/security/pam_sss.so

# hadolint ignore=DL3041
RUN dnf -y install bash coreutils diffutils net-tools htop openssh-server sssd \
  sudo dnf-plugins-core tree git wget which ripgrep fzf procps-ng && \
  dnf clean all

# add user/group
# RUN groupadd mock
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
RUN adduser -g wheel packager && \
  echo "root:${ROOT_PWD}" | chpasswd && \
  echo "packager:${USER_PWD}" | chpasswd && \
  # add packager to sudo list
  echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager && \
  # start automatically at the boot time
  systemctl enable sshd.service && \
  sed -i \
  -e 's/#PermitRootLogin.*/PermitRootLogin\ yes/g' \
  -e 's/#LogLevel.*/LogLevel\ VERBOSE/g' \
  -e 's/#PubkeyAuthentication.*/PubkeyAuthentication\ yes/g' \
  -e 's/#Port 22/Port 22/g' \
  /etc/ssh/sshd_config

# https://developer.fedoraproject.org/deployment/rpm/about.html
EXPOSE 22
EXPOSE 8101/udp
EXPOSE 8102/udp
EXPOSE 8103/udp

ENTRYPOINT ["/sbin/init"]
