# The API container no longer links against libtesseract — OCR runs in the
# sandboxed ocr-service container. That lets us build a fully static binary and
# ship it on distroless/static, eliminating the shell, package manager, and
# every C library that previously sat in this image.

FROM golang:1.24-alpine@sha256:ef18ee7117463ac1055f5a370ed18b8750f01589f13ea0b48642f5792b234044 AS build-env

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . /app

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/mqtt ./main.go

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build-env /out/mqtt /app/mqtt

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/mqtt"]
