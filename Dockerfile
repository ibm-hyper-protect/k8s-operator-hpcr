FROM golang as build_layer

COPY . /src

RUN cd /src && \
    mkdir -p /build && \
    go build -o /build/hpcr-controller -v main.go 

FROM registry.access.redhat.com/ubi8/ubi-minimal as base_layer

FROM scratch

COPY --from=base_layer /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=base_layer /etc/pki/tls/ /etc/pki/tls/
COPY --from=base_layer /etc/pki/ca-trust/ /etc/pki/ca-trust/

COPY --from=build_layer /build/hpcr-controller /hpcr-controller

ENTRYPOINT [ "/hpcr-controller" ]