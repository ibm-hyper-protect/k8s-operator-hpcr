FROM registry.access.redhat.com/ubi8/ubi-minimal as base_layer

FROM scratch

COPY --from=base_layer /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=base_layer /etc/pki/tls/ /etc/pki/tls/
COPY --from=base_layer /etc/pki/ca-trust/ /etc/pki/ca-trust/

COPY build/hpcr-controller /hpcr-controller

EXPOSE 8080

ENTRYPOINT [ "/hpcr-controller" ]