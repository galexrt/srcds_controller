FROM galexrt/gameserver:v20201022-132454-692
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

# Copied from https://github.com/mono/docker/blob/a449b2a57e1cfadd098c2bcad13f89c4eab83e54/6.10.0.104/slim/Dockerfile
# till per gameserver images are available

# COPY BEGIN
ENV MONO_VERSION 6.10.0.104

RUN apt-get update \
  && apt-get install -y --no-install-recommends gnupg dirmngr \
  && rm -rf /var/lib/apt/lists/* \
  && export GNUPGHOME="$(mktemp -d)" \
  && gpg --batch --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 3FA7E0328081BFF6A14DA29AA6A19B38D3D831EF \
  && gpg --batch --export --armor 3FA7E0328081BFF6A14DA29AA6A19B38D3D831EF > /etc/apt/trusted.gpg.d/mono.gpg.asc \
  && gpgconf --kill all \
  && rm -rf "$GNUPGHOME" \
  && apt-key list | grep Xamarin \
  && apt-get purge -y --auto-remove gnupg dirmngr

RUN echo "deb http://download.mono-project.com/repo/debian stable-buster/snapshots/$MONO_VERSION main" > /etc/apt/sources.list.d/mono-official-stable.list \
  && apt-get update \
  && apt-get install -y mono-runtime \
  && rm -rf /var/lib/apt/lists/* /tmp/*
# COPY END

ADD .build/linux-amd64/srcds_runner /bin/srcds_runner

RUN chmod 755 /bin/srcds_runner

ENTRYPOINT ["/tini", "-s", "--", "/bin/srcds_runner"]
