FROM alpine:3.20
LABEL maintainer="ericwq057@qq.com"
# build_date="2024-06-28"

# hadolint ignore=DL3018,DL3004
RUN apk add --no-cache alpine-sdk sudo mandoc abuild-doc tzdata atools && \
  adduser -D -G abuild packager && \
  addgroup packager abuild && \
  echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager && \
  echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories

USER packager
WORKDIR /home/packager

# hadolint ignore=DL3004 it's working
RUN sudo -u packager abuild-keygen -n --append --install

# hadolint ignore=DL3059 it's take a lot time to be done
RUN git clone https://gitlab.alpinelinux.org/ericwq057/aports.git

CMD ["/bin/ash"]
