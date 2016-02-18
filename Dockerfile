FROM scratch
MAINTAINER Getty Images "https://github.com/gettyimages"

ADD bin/chronos_exporter /chronos_exporter
ENTRYPOINT ["/chronos_exporter"]

EXPOSE 9044
