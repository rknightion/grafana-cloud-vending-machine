# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

ARG GO_VERSION=1.27.3
ARG GO_IMAGE_DIGEST=sha256:b76020929b709ddb6202afc05410a0faefae20ae05e27ad548b3870092688a69
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}@${GO_IMAGE_DIGEST} AS build

WORKDIR /function
ENV CGO_ENABLED=0

RUN --mount=target=. --mount=type=cache,target=/go/pkg/mod go mod download

ARG TARGETOS
ARG TARGETARCH
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /function-server .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99 AS image
COPY --from=build /function-server /function-server
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT ["/function-server"]
