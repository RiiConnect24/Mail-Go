# Mail-Go
[![License](https://img.shields.io/github/license/riiconnect24/mail-go.svg?style=flat-square)](http://www.gnu.org/licenses/agpl-3.0)
![Production List](https://img.shields.io/discord/206934458954153984.svg?style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/RiiConnect24/Mail-Go?style=flat-square)](https://goreportcard.com/report/github.com/RiiConnect24/Mail-Go)

This is an effort to rewrite Wii Mail legacy PHP scripts into golang.
Some reasons why:
- `apache2` has the fun tendency to go overboard on memory usage.
- `go` is fun.

# How to develop
The source is entirely here, with each individual cgi component in their own file.
A `Dockerfile` is available to create an image. You can use `docker-compose.yml` to develop on this specific component with its own mysql, or use *something that doesn't yet exist* to develop on RC24 as a whole.
You can use `docker-compose up` to start up both MariaDB and Mail-Go.

# How can I use the patcher for my own usage?
You're welcome to `POST /patch` with a `nwc24msg.cfg` under form key `uploaded_config`.

# What should I do if I'm adding a new dependency?
There's a `get.sh` script that has all major external dependencies. This allows us to cache `go get`.
If you're adding another dependency, it's recommended you add that to the script.