package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"log"
	"net/http"
	"strconv"
)

const SuccessfulResponse = "cd=100\n" + "msg=Success.\n"

var (
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	reset   = string([]byte{27, 91, 48, 109})
	warning = red + "[Warning]" + reset + "Encountered error %d with reason %s"
)

func TypedErrorResponse(c *gin.Context, code int, separator string, reason string) {
	if code != 220 {
		log.Printf(warning, code, reason)
	}

	c.Render(http.StatusOK, render.String{Format: fmt.Sprintf("cd%s%d\nmsg%s%s\n", separator, code, separator, reason), Data: nil})
}

func ErrorResponse(c *gin.Context, code int, reason string) {
	if code != 220 {
		log.Printf(warning, code, reason)
	}

	TypedErrorResponse(c, code, "=", reason)
}

func MailErrorResponse(code int, reason string, mailNumber string) string {
	if code != 100 {
		log.Printf(warning, code, reason)
	}

	return fmt.Sprint("cd", mailNumber[1:], "=", strconv.Itoa(code), "\n",
		"msg", mailNumber[1:], "=", reason, "\n")
}
