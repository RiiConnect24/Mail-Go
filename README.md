# Mail-Go
[![Go Report Card](https://goreportcard.com/badge/github.com/RiiConnect24/Mail-Go)](https://goreportcard.com/report/github.com/RiiConnect24/Mail-Go)

This is an effort to rewrite soon-to-be-legacy PHP scripts into golang.
Some reasons why:
- `apache2` has the fun tendency to go overboard on memory usage.
- `go` is fun.

# How to develop
The source is entirely here, with each individual cgi component in their own file.
A `Dockerfile` is available to create an image. You can use `docker-compose.yml` to develop on this specific component with its own mysql, or use *something that doesn't yet exist* to develop on RC24 as a whole.
You can use `docker-compose up` to start up both MySQL and Mail-Go.

# How can I use the patcher for my own usage?
You're welcome to `POST /patch` with a `nwc24msg.cfg` under form key `uploaded_config`.