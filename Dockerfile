FROM ubuntu

COPY _out/bin/vega /usr/local/bin/vega

ENTRYPOINT ["/usr/local/bin/vega"]
