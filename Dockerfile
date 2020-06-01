FROM galexrt/gameserver:v20200601-144434-783
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner

RUN chmod 755 /bin/srcds_runner

ENTRYPOINT ["/tini", "-s", "--", "/bin/srcds_runner"]
