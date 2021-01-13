FROM galexrt/gameserver:v20210113-183742-977
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner
ADD .build/linux-amd64/sc /bin/sc
ADD .build/linux-amd64/srcds_controller /bin/srcds_controller

RUN chmod 755 /bin/srcds_runner /bin/sc /bin/srcds_controller

ENTRYPOINT ["/tini", "-s", "--", "/bin/srcds_runner"]
