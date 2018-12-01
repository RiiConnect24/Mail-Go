#!/bin/sh
# Script allowing us to cache Go dependencies in Docker cache.
go get -v github.com/google/uuid
go get -v github.com/logrusorgru/aurora
go get -v github.com/go-sql-driver/mysql
go get -v github.com/getsentry/raven-go
go get -v github.com/robfig/cron

# Used for image processing in regards to e-mail
go get -v github.com/nfnt/resize
go get -v golang.org/x/image/bmp
go get -v golang.org/x/image/tiff
go get -v golang.org/x/image/webp