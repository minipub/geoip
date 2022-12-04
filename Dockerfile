FROM kakie/golang:latest as BaseBuilder
COPY . /opt/geoip/
WORKDIR /opt/geoip
RUN go build

FROM alpine:3.17
COPY --from=BaseBuilder /opt/geoip/geoip /usr/local/bin/
CMD ["/usr/local/bin/geoip", "8080"]
