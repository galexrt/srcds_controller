FROM ghcr.io/galexrt/gameserver:v20210919-000315-232

ARG BUILD_DATE="N/A"
ARG REVISION="N/A"

LABEL org.opencontainers.image.authors="Alexander Trost <galexrt@googlemail.com>" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.title="galexrt/srcds_controller" \
    org.opencontainers.image.description="N/A" \
    org.opencontainers.image.documentation="https://github.com/galexrt/srcds_controller/blob/main/README.md" \
    org.opencontainers.image.url="https://github.com/galexrt/srcds_controller" \
    org.opencontainers.image.source="https://github.com/galexrt/srcds_controller" \
    org.opencontainers.image.revision="${REVISION}" \
    org.opencontainers.image.vendor="galexrt" \
    org.opencontainers.image.version="N/A"

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner
ADD .build/linux-amd64/sc /bin/sc
ADD .build/linux-amd64/srcds_controller /bin/srcds_controller

RUN chmod 755 /bin/srcds_runner /bin/sc /bin/srcds_controller

ENTRYPOINT ["/tini", "-s", "--", "/bin/srcds_runner"]
