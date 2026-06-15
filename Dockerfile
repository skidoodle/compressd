FROM ubuntu:26.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    curl \
    git \
    libvips-dev \
    libheif-dev \
    libwebp-dev \
    patchelf \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L https://go.dev/dl/go1.26.4.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

COPY . .

RUN export CGO_ENABLED=1 && \
    go build -v -o compressd .

RUN chmod +x bundle_linux.sh && \
    ./bundle_linux.sh compressd

RUN find lib/ -name "*heif*" | grep heif || (echo "FAILED: libheif not found in bundled libs" && vips -l | grep heif && ldd compressd && exit 1)

CMD ["go", "test", "-v", "./engine/..."]
