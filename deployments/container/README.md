# container

This directory contains support files for running `registry-server` in a
container with and without an Envoy proxy.

Use `Dockerfile.noproxy` to build a container with `registry-server` only.

Use `Dockerfile.envoy` to build a container with `registry-server` and Envoy.
Envoy is configured to support grpc-web and to perform authorization using the
`authz-server` that is included with this project, which is also included and
run in the container.
