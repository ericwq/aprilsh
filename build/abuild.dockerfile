FROM alpine:3.19
LABEL maintainer="ericwq057@qq.com"

#
RUN apk add --no-cache --update alpine-sdk sudo mandoc abuild-doc tzdata atools
RUN adduser -D packager
RUN addgroup packager abuild
RUN echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
RUN sudo -u packager abuild-keygen -n --append --install
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories

USER packager:abuild
RUN cd ~ && \
	# mkdir -p packages/testing/ packages/main/ packages/community/ && \
	git clone https://gitlab.alpinelinux.org/ericwq057/aports.git

# USER root

# ENV PATH=$OLDPATH
CMD ["/bin/ash"]
