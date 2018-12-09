FROM golang:1.11-alpine3.8 as builder

# We assume only git is needed for all dependencies.
# openssl is already built-in.
RUN apk add -U --no-cache git

WORKDIR /go/src/github.com/RiiConnect24/Mail-Go
COPY get.sh /go/src/github.com/RiiConnect24/Mail-Go
RUN sh get.sh

# Copy necessary parts of the Mail-Go source into builder's source
COPY *.go ./
COPY patch patch

# Build to name "app".
RUN go build -o app .

###########
# RUNTIME #
###########
FROM alpine:3.8

WORKDIR /go/src/github.com/RiiConnect24/Mail-Go/
COPY --from=builder /go/src/github.com/RiiConnect24/Mail-Go/ .

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz && apk add -U --no-cache ca-certificates

# Wait until there's an actual MySQL connection we can use to start.
ENTRYPOINT ["/go/src/github.com/RiiConnect24/Mail-Go/app"]
