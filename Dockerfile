FROM mirror.gcr.io/alpine:latest

ADD /bin/rag-server /app/bin/rag-server
WORKDIR /app

EXPOSE 8080
CMD [ "/app/bin/rag-server" ]
