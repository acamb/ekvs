# ─── Stage: generate ed25519 key WITHOUT passphrase ──────────────────────────
FROM alpine:latest AS keygen-nopass
RUN apk add --no-cache openssh-keygen && \
    mkdir /keys && \
    ssh-keygen -t ed25519 -N "" -f /keys/id_ed25519

# ─── Stage: generate ed25519 key WITH passphrase ─────────────────────────────
FROM alpine:latest AS keygen-passphrase
RUN apk add --no-cache openssh-keygen && \
    mkdir /keys && \
    ssh-keygen -t ed25519 -N "changeme" -f /keys/id_ed25519

# ─── Stage: build server ──────────────────────────────────────────────────────
FROM golang:1.25-alpine AS server-build
WORKDIR /src
COPY . .
RUN go build -o /server ./cmd/server

# ─── Stage: build cli ─────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS cli-build
WORKDIR /src
COPY . .
RUN go build -o /cli ./cmd/cli

# ─── Stage: build tui ─────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS tui-build
WORKDIR /src
COPY . .
RUN go build -o /tui ./cmd/tui

# ─── Runtime: server (nopass) ─────────────────────────────────────────────────
FROM alpine:latest AS server-nopass
COPY --from=server-build /server /server
RUN mkdir -p /data/.keys
COPY --from=keygen-nopass /keys/id_ed25519.pub /data/.keys/testuser.pub
RUN chmod 0644 /data/.keys/testuser.pub && \
    printf 'server_addr:  "0.0.0.0:8080"\nstorage_path: "/data"\nkeys_dir:     "/data/.keys"\nlog_level:    "debug"\n' > /ekvs.yaml
WORKDIR /
EXPOSE 8080
CMD ["/server", "--config", "/ekvs.yaml"]

# ─── Runtime: server (passphrase) ────────────────────────────────────────────
FROM alpine:latest AS server-passphrase
COPY --from=server-build /server /server
RUN mkdir -p /data/.keys
COPY --from=keygen-passphrase /keys/id_ed25519.pub /data/.keys/testuser.pub
RUN chmod 0644 /data/.keys/testuser.pub && \
    printf 'server_addr:  "0.0.0.0:8080"\nstorage_path: "/data"\nkeys_dir:     "/data/.keys"\nlog_level:    "debug"\n' > /ekvs.yaml
WORKDIR /
EXPOSE 8080
CMD ["/server", "--config", "/ekvs.yaml"]

# ─── Runtime: cli (nopass) ────────────────────────────────────────────────────
FROM alpine:latest AS cli-nopass
COPY --from=cli-build /cli /cli
RUN mkdir -p /root/.ssh
COPY --from=keygen-nopass /keys/id_ed25519 /root/.ssh/id_ed25519
RUN chmod 0600 /root/.ssh/id_ed25519
ENV EKVS_SERVER=http://server:8080
ENV EKVS_IDENTITY=/root/.ssh/id_ed25519
WORKDIR /

# ─── Runtime: cli (passphrase) ───────────────────────────────────────────────
FROM alpine:latest AS cli-passphrase
COPY --from=cli-build /cli /cli
RUN mkdir -p /root/.ssh
COPY --from=keygen-passphrase /keys/id_ed25519 /root/.ssh/id_ed25519
RUN chmod 0600 /root/.ssh/id_ed25519
ENV EKVS_SERVER=http://server:8080
ENV EKVS_IDENTITY=/root/.ssh/id_ed25519
ENV EKVS_PASSPHRASE=changeme
WORKDIR /

# ─── Runtime: tui (nopass) ────────────────────────────────────────────────────
FROM alpine:latest AS tui-nopass
COPY --from=tui-build /tui /tui
RUN mkdir -p /root/.ssh
COPY --from=keygen-nopass /keys/id_ed25519 /root/.ssh/id_ed25519
RUN chmod 0600 /root/.ssh/id_ed25519 && \
    printf 'profiles:\n  - name:          "integration"\n    server_url:    "http://server:8080"\n    identity_file: "/root/.ssh/id_ed25519"\n    theme:         "adaptive"\n' > /ekvs-tui.yaml
WORKDIR /

# ─── Runtime: tui (passphrase) ───────────────────────────────────────────────
FROM alpine:latest AS tui-passphrase
COPY --from=tui-build /tui /tui
RUN mkdir -p /root/.ssh
COPY --from=keygen-passphrase /keys/id_ed25519 /root/.ssh/id_ed25519
RUN chmod 0600 /root/.ssh/id_ed25519 && \
    printf 'profiles:\n  - name:          "integration"\n    server_url:    "http://server:8080"\n    identity_file: "/root/.ssh/id_ed25519"\n    theme:         "adaptive"\n' > /ekvs-tui.yaml
WORKDIR /

