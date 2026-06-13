FROM mirror.gcr.io/debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-suggests --no-install-recommends ca-certificates curl && \
    apt-get clean && \
    groupadd -g 1001 microuser && \
    useradd -u 1001 -r -g 1001 -s /sbin/nologin -c "go microservice user" microuser

RUN update-ca-certificates --fresh

ADD ./bin/rag-server /app/bin/
WORKDIR /app

EXPOSE 8080

USER microuser
ENTRYPOINT [ "/app/bin/rag-server" ]
