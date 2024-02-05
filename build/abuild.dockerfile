FROM alpine:3.19
LABEL maintainer="ericwq057@qq.com"

#
RUN apk add --no-cache --update alpine-sdk sudo mandoc abuild-doc tzdata
RUN adduser -D packager
RUN addgroup packager abuild
RUN echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
RUN sudo -u packager abuild-keygen -n --append --install

USER packager:abuild
RUN cd ~ && mkdir -p aports/main/aprilsh && cd ~/aports/main/aprilsh/

USER root

# ENV PATH=$OLDPATH
CMD ["/bin/ash"]
