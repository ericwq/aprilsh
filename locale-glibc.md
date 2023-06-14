## glibc locales

see [here](http://blog.fpliu.com/it/software/GNU/glibc#alpine) and [there](https://zhuanlan.zhihu.com/p/151852282) for install glibc in alpine linux.

```sh
% curl -L -o /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
% export APK_GLIBC_VERSION=2.35-r0
% export APK_GLIBC_BASE_URL="https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${APK_GLIBC_VERSION}"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-bin-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-dev-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-i18n-${APK_GLIBC_VERSION}.apk"
% apk add glibc-${APK_GLIBC_VERSION}.apk glibc-bin-${APK_GLIBC_VERSION}.apk glibc-dev-${APK_GLIBC_VERSION}.apk glibc-i18n-${APK_GLIBC_VERSION}.apk
% rm glibc-*
% export PATH=/usr/glibc-compat/bin:$PATH
```

Intall required locale.

```sh
% localedef -i zh_CN -f GB18030 zh_CN.GB18030
% localedef -i en_US -f UTF-8 en_US.UTF-8
```

check [here](https://gist.github.com/larzza/0f070a1b61c1d6a699653c9a792294be) for install glibc in alpine docker image.
