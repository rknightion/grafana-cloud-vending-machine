FROM ghcr.io/rknightion/cursor-cloud-base:1@sha256:d682eea83896694ce529c2035816610edd723c9acbd39ccaf8056743e7ed1756

USER root
# The repository's manifest validation runs Ruby and kubectl.
RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
        amd64) archive=https://archive.ubuntu.com/ubuntu/; security=https://security.ubuntu.com/ubuntu/ ;; \
        arm64) archive=https://ports.ubuntu.com/ubuntu-ports/; security="$archive" ;; \
        *) exit 1 ;; \
    esac; \
    key=/usr/share/keyrings/ubuntu-archive-keyring.gpg; \
    printf 'deb [signed-by=%s] %s noble main\ndeb [signed-by=%s] %s noble-updates main\ndeb [signed-by=%s] %s noble-security main\n' \
        "$key" "$archive" "$key" "$archive" "$key" "$security" > /tmp/cursor-ruby.list; \
    set -- -o Dir::Etc::sourcelist=/tmp/cursor-ruby.list -o Dir::Etc::sourceparts=-; \
    apt-get "$@" -o APT::Update::Error-Mode=any update; \
    DEBIAN_FRONTEND=noninteractive apt-get "$@" install -y --no-install-recommends ruby; \
    ruby -ryaml -e 'YAML.safe_load("ready: true")'; \
    rm -rf /var/lib/apt/lists/* /tmp/cursor-ruby.list

RUN set -eu; \
    version=v1.37.0; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
        amd64) digest=6129359f4e1f3848a5572ccb0b26cf28b8ca08cef38c95a765b2f64a2c961a2f ;; \
        arm64) digest=922df28df248cc00a9e025f947704f1d1482de64ece54cfe57e61f19eaf1eef3 ;; \
        *) exit 1 ;; \
    esac; \
    binary="$(mktemp)"; \
    curl -qfsSL --retry 2 --max-time 180 -o "$binary" \
        "https://dl.k8s.io/release/$version/bin/linux/$arch/kubectl"; \
    printf '%s  %s\n' "$digest" "$binary" | sha256sum --check --status; \
    install -m 0755 "$binary" /usr/local/bin/kubectl; \
    rm -f "$binary"

ENV KUBECONFIG=/dev/null
USER ubuntu
