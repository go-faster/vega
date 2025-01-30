FROM ubuntu

COPY _out/bin/vega-ingest /usr/local/bin/vega-ingest

ENTRYPOINT ["/usr/local/bin/vega-ingest"]
