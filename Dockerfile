FROM galexrt/gameserver:latest
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner

RUN chmod 755 /bin/srcds_runner

ENTRYPOINT ["/tini", "--", "/bin/srcds_runner"]
