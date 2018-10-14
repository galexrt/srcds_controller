FROM quay.io/prometheus/busybox:latest
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_controller /bin/srcds_controller

ENTRYPOINT ["/bin/srcds_controller"]
