# Build stage: full toolchain + libtesseract/leptonica headers for CGO link.
FROM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
        libtesseract-dev \
        libleptonica-dev \
        pkg-config \
        tesseract-ocr-eng \
        tesseract-ocr-ron \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/ocr-service .

# Resolve the dynamic libraries the binary actually loads at runtime and stage
# them so we only carry what's needed into the distroless image.
RUN set -eux; \
    mkdir -p /stage/usr/lib/x86_64-linux-gnu /stage/lib64; \
    ldd /out/ocr-service | awk '/=>/ {print $3}' | sort -u | while read -r lib; do \
        [ -e "$lib" ] && cp -L "$lib" /stage/usr/lib/x86_64-linux-gnu/; \
    done; \
    cp -L /lib64/ld-linux-x86-64.so.2 /stage/lib64/ 2>/dev/null || true

# Runtime stage: distroless/cc provides glibc + libgcc + libstdc++ on a base
# with no shell, no package manager, and no other binaries. Runs as the
# pre-baked `nonroot` user (uid 65532).
FROM gcr.io/distroless/cc-debian12:nonroot

# Tesseract shared libs + transitive deps discovered above.
COPY --from=builder /stage/ /

# Trained language data — only the languages we actually use.
COPY --from=builder /usr/share/tesseract-ocr/ /usr/share/tesseract-ocr/

# The Go service binary.
COPY --from=builder /out/ocr-service /ocr-service

ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/5/tessdata/

USER nonroot:nonroot
EXPOSE 9090
ENTRYPOINT ["/ocr-service"]
