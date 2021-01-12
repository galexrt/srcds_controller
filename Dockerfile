FROM galexrt/gameserver:v20210107-130805-533
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner
ADD .build/linux-amd64/sc /bin/sc
ADD .build/linux-amd64/srcds_controler /bin/srcds_controler

RUN chmod 755 /bin/srcds_runner /bin/sc /bin/srcds_controler

ENTRYPOINT ["/tini", "-s", "--", "/bin/srcds_runner"]
