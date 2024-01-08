FROM golang:1.21.5-alpine3.18 as build

WORKDIR /app
COPY . /app

RUN apk add --no-cache make
RUN make build

FROM alpine:3.18
COPY --from=build /app/bin/ /app/bin/

CMD ["/app/bin/proclet", "server"]
