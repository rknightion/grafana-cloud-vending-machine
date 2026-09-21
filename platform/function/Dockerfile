# syntax=docker/dockerfile:1@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32

ARG GO_VERSION=1.27.1
ARG GO_IMAGE_DIGEST=sha256:512690a5660563b57d37ecc31129e7f136e831db2aed24a1dbeb8ad7380dc0fa
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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS image
COPY --from=build /function-server /function-server
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT ["/function-server"]
