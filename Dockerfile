FROM scratch

VOLUME /etc/nginx/
EXPOSE 8080 9090

ARG ARCH=""

COPY bin/doorman${ARCH} /doorman

ENV HOME=/var/www

ENTRYPOINT ["/doorman"]
