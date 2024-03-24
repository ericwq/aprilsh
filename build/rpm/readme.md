## prepare container for rpm package

create container according to [RPM Packaging Guide](https://rpm-packaging-guide.github.io/#introduction). The container insatll `gcc rpm-build rpm-devel rpmlint make python bash coreutils diffutils patch rpmdevtools` packages. create `packager` user.

```sh
docker build -t rpm-builder:0.1.0 -f centos7.dockerfile .
docker build --no-cache --progress plain -t rpm-builder:0.1.0 -f centos7.dockerfile .
```

run as packager
```sh
docker run -u packager --rm -ti -h rpm-builder --env TZ=Asia/Shanghai --name rpm-builder --privileged \
        --mount source=proj-vol,target=/home/packager/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/packager/develop \
        rpm-builder:0.1.0
```

## build rpm package

```sh
rpmdev-setuptree
cd rpmbuild/SPECS
rpmbuild -ba hello-world.spec
```

## install go with yum
For installing latest go versions, you would need to add a repository with the latest versions.
```sh
sudo rpm --import https://mirror.go-repo.io/centos/RPM-GPG-KEY-GO-REPO
curl -s https://mirror.go-repo.io/centos/go-repo.repo | sudo tee /etc/yum.repos.d/go-repo.repo
sudo yum install golang
```

## install go with dnf
[How To Install Go (Golang) On Fedora](https://computingforgeeks.com/how-to-install-go-golang-on-fedora/)
```sh 
sudo dnf -y update
sudo dnf install go
```
