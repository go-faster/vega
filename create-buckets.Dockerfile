FROM ubuntu

COPY _out/bin/create-buckets /usr/local/bin/create-buckets

ENTRYPOINT ["/usr/local/bin/create-buckets"]
