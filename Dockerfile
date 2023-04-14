# Copyright 2023 IBM Corp.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.package datasource
FROM registry.access.redhat.com/ubi8/ubi-minimal as base_layer

RUN mkdir -p /base_tmp/

FROM scratch

COPY --from=base_layer /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=base_layer /etc/pki/tls/ /etc/pki/tls/
COPY --from=base_layer /etc/pki/ca-trust/ /etc/pki/ca-trust/

COPY --from=base_layer /base_tmp/ /tmp/

COPY k8s-operator-hpcr /k8s-operator-hpcr

EXPOSE 8080

ENTRYPOINT [ "/k8s-operator-hpcr" ]