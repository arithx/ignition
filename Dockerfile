FROM golang:_GOVERSION_

RUN echo "deb http://deb.debian.org/debian stretch main" >> /etc/apt/sources.list
RUN echo "deb http://us.archive.ubuntu.com/ubuntu/ trusty main " >> /etc/apt/sources.list

# gcc for cgo
RUN apt-get update && apt-get install --force-yes -y \
		g++ \
		gcc-4.8 \
		libc6-dev \
		make \
		pkg-config \
		gcc-4.8-aarch64-linux-gnu \
		libc6-dev-arm64-cross \
		libblkid-dev \
		sudo \
		uuid-runtime \
		gdisk \
		kpartx \
		e2fsprogs \
		dosfstools \
		file \
	&& rm -rf /var/lib/apt/lists/*

ENV GOLANG_VERSION _GOVERSION_

RUN echo '%jenkins ALL=NOPASSWD:ALL' >> /etc/sudoers;

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
WORKDIR $GOPATH
