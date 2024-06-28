FROM alpine:3.20
LABEL maintainer="ericwq057@qq.com"
# build_date="2024-06-28"

# hadolint ignore=DL3018
RUN apk add --no-cache alpine-sdk sudo mandoc abuild-doc tzdata atools && \
  adduser -D packager && \
  addgroup packager abuild && \
  echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager && \
  echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories

USER packager:abuild
WORKDIR /home/packager

RUN  abuild-keygen -n --append --install && \
  # mkdir -p packages/testing/ packages/main/ packages/community/ && \
  git clone https://gitlab.alpinelinux.org/ericwq057/aports.git

CMD ["/bin/ash"]
