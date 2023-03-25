FROM gcr.io/distroless/static

ADD testbot /usr/local/bin/testbot

ENTRYPOINT ["testbot"]
