FROM alpine:latest
WORKDIR /ekvs
COPY bin/server-static ./server
RUN mkdir -p data/.keys/
COPY ekvs.yaml.example ./ekvs.yaml
CMD ["./server", "--config", "/ekvs/ekvs.yaml"]
