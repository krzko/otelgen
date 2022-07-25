FROM scratch
ENTRYPOINT ["/otelgen"]
COPY proto2yaml /otelgen
