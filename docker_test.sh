sudo yum update
sudo yum install gcc-aarch64-linux-gnu libc6-dev-arm64-cross libblkid-dev kpartx gdisk

git clone https://github.com/coreos/ignition

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

if [ "${TARGET}" == "amd64" ]; then
      GOARCH="${TARGET}" sudo -E env "PATH=$PATH" ./test;
elif [ "${TARGET}" == "arm64" ]; then
      export CGO_LDFLAGS="-L ${PWD}";
      GOARCH="${TARGET}" ./build;
      file "bin/${TARGET}/ignition" | egrep 'aarch64';
fi
