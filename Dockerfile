FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
ENV GO_VERSION=1.26.4

RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    curl \
    git \
    libvips-dev \
    libheif-dev \
    libheif-plugin-aomdec \
    libheif-plugin-aomenc \
    libheif-plugin-dav1d \
    libheif-plugin-libde265 \
    libheif-plugin-x265 \
    libwebp-dev \
    patchelf \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf -
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

ENV VIPS_MODULE_PATH=/app/lib

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN export CGO_ENABLED=1 && \
    go build -v -ldflags="-s -w" -o compressd .

RUN chmod +x bundle_linux.sh && \
    ./bundle_linux.sh compressd

RUN ls -R lib/ && \
    ldd lib/vips-modules-8.15/vips-heif.so && \
    if [ -d lib/libheif ]; then ls -R lib/libheif; fi && \
    ./compressd --version

FROM ubuntu:24.04 AS tester

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /test

COPY --from=builder /app/compressd .
COPY --from=builder /app/lib ./lib

RUN echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" | base64 -d > test.png

RUN VIPS_DEBUG=1 ./compressd --version && \
    VIPS_DEBUG=1 ./compressd -v -f avif -e . || (echo "FAILED: AVIF support missing in bundled binary" && ls -R lib/ && exit 1) && \
    [ -f test.avif ] && echo "SUCCESS: AVIF conversion verified"

CMD ["./compressd", "--help"]
