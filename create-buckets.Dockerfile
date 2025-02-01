FROM ubuntu

COPY _out/bin/vega-create-buckets /usr/local/bin/vega-create-buckets

ENTRYPOINT ["/usr/local/bin/vega-create-buckets"]
