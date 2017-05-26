FROM golang:_GOVERSION_

RUN echo "deb http://deb.debian.org/debian stretch main" >> /etc/apt/sources.list

# gcc for cgo
RUN apt-get update && apt-get install --force-yes -y --no-install-recommends \
		g++ \
		gcc \
		libc6-dev \
		make \
		pkg-config \
		gcc-aarch64-linux-gnu \
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
