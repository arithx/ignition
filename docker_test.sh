source /env_vars.sh

sudo yum update

sudo curl -sL -o /bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
sudo chmod +x /bin/gimme

echo $GIMME_GO_VERSION
#eval "$(gimme $GIMME_GO_VERSION)"
eval "$(gimme 1.7)"

sudo yum install git gcc-aarch64-linux-gnu libc6-dev-arm64-cross libblkid-devel kpartx gdisk -y
sudo yum group install "Development Tools" -y

export GOPATH="$(pwd)/go"

mkdir go
mkdir go/src
mkdir go/src/github.com
mkdir go/src/github.com/coreos

cd go/src/github.com/coreos

ls /
ls /ignition
mv /ignition -t .
#git clone https://github.com/coreos/ignition

cd ignition

# since libblkid-dev:arm64 cannot be installed, spoof it
if [ "${TARGET}" == "arm64" ]; then
      echo "void blkid_new_probe_from_filename(void) {}" >> stub.c;
      echo "void blkid_do_probe(void) {}"                >> stub.c;
      echo "void blkid_probe_has_value(void) {}"         >> stub.c;
      echo "void blkid_probe_lookup_value(void) {}"      >> stub.c;
      echo "void blkid_free_probe(void) {}"              >> stub.c;
      aarch64-linux-gnu-gcc-4.8 -c -o stub.o stub.c;
      aarch64-linux-gnu-gcc-ar-4.8 cq libblkid.a stub.o;
fi

echo $TARGET

if [ "${TARGET}" == "amd64" ]; then
      GOARCH="${TARGET}" sudo -E env "PATH=$PATH:/bin/gcc" ./test;
elif [ "${TARGET}" == "arm64" ]; then
      export CGO_LDFLAGS="-L ${PWD}";
      GOARCH="${TARGET}" ./build;
      file "bin/${TARGET}/ignition" | egrep 'aarch64';
fi
