##################################################################
# Build binary.
##################################################################
FROM golang:1.17.3 as build
WORKDIR /go/src/github.com/mslocrian/avi_exporter
ADD . .
RUN make build-local

##################################################################\
# Build non-root container.
##################################################################
FROM alpine:latest
EXPOSE 9300
RUN addgroup -g 1000 aviexporter &&\
    adduser aviexporter -D aviexporter -u 1000 -G aviexporter
USER 1000:1000
COPY --from=build /go/src/github.com/mslocrian/avi_exporter/avi_exporter /home/aviexporter/avi_exporter
COPY --from=build /go/src/github.com/mslocrian/avi_exporter/lib /home/aviexporter/lib
WORKDIR /home/aviexporter
ENTRYPOINT ["/home/aviexporter/avi_exporter"]
