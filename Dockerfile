FROM golang:1.17-alpine3.14 as builder

# We assume only git is needed for all dependencies.
# openssl is already built-in.
RUN apk add -U --no-cache git

WORKDIR /go/src/github.com/RiiConnect24/Mail-Go

# Cache pulled dependencies if not updated.
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy necessary parts of the Mail-Go source into builder's source
COPY *.go ./
COPY patch patch

# Build to name "app".
RUN go build -o app .

###########
# RUNTIME #
###########
FROM alpine:3.14

WORKDIR /go/src/github.com/RiiConnect24/Mail-Go/

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz && apk add -U --no-cache ca-certificates

COPY --from=builder /go/src/github.com/RiiConnect24/Mail-Go/ .

# Wait until there's an actual MySQL connection we can use to start.
CMD ["dockerize", "-wait", "tcp://127.0.0.1:3306", "-timeout", "60s", "/go/src/github.com/RiiConnect24/Mail-Go/app"]