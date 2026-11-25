# syntax=docker/dockerfile:1@sha256:572cebbc2b15e6f84c077742952674889cb63cd5e9b80abdffcbe587b6f19183

ARG GO_VERSION=1.27.4
ARG GO_IMAGE_DIGEST=sha256:5f1b98fa02fe396ebce2a0227a0e08309e11792915b9dd2e1f71053d7e7e6666
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

FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5 AS image
COPY --from=build /function-server /function-server
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT ["/function-server"]
