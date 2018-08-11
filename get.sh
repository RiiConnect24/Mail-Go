#!/bin/sh
# Script allowing us to cache Go dependencies in Docker cache.
go get github.com/google/uuid
go get github.com/discordapp/lilliput
go get github.com/logrusorgru/aurora
go get github.com/go-sql-driver/mysql
go get github.com/getsentry/raven-go