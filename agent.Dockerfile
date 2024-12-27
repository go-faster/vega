FROM ubuntu

COPY _out/bin/vega-agent /usr/local/bin/vega-agent

ENTRYPOINT ["/usr/local/bin/vega-agent"]
