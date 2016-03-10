FROM scratch

COPY config-bob /usr/bin/config-bob

ENTRYPOINT ["/usr/bin/config-bob"]
