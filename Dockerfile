FROM scratch

VOLUME /etc/nginx/
EXPOSE 8080 9090

ARG ARCH=""

COPY bin/doorman${ARCH} /doorman

ENTRYPOINT ["/doorman"]
