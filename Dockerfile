FROM golang:1.10

# For later timing purposes
RUN apt-get update && apt-get install -y wget

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    # Clean packages
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz && apt-get clean

# Pre-download listed dependencies to take
# advantage of Docker cache.
RUN mkdir -p /go/src/Mail-Go
WORKDIR /go/src/Mail-Go
COPY get.sh /go/src/Mail-Go
RUN sh get.sh

# Copy the entire Mail-Go source into builder's source.
COPY . .
RUN go get ./...

# Build to name "app".
RUN GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o app .

# Wait until there's an actual MySQL connection we can use to start.
# CMD ["dockerize", "-wait", "tcp://database:3306", "-timeout", "60s", "/go/src/Mail-Go/app"]
CMD ["/app"]