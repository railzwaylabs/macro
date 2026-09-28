FROM scratch

ARG TARGETPLATFORM

COPY ${TARGETPLATFORM}/macro /usr/local/bin/macro

ENTRYPOINT ["/usr/local/bin/macro"]
