package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/winebarrel/cronplan"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		host := c.Request.Host
		c.String(http.StatusOK, fmt.Sprintf("USAGE:\ncurl %s -d '5 0 * * ? *'\n", host))
	})

	r.POST("/", func(c *gin.Context) {
		exp, err := io.ReadAll(c.Request.Body)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		cron, err := cronplan.Parse(string(exp))

		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		triggers := cron.NextN(time.Now(), 10)
		var buf strings.Builder

		for _, t := range triggers {
			buf.WriteString(t.Format("Mon, 02 Jan 2006 15:04:05 MST"))
			buf.WriteString("\n")
		}

		c.String(http.StatusOK, buf.String())
	})

	r.Run()
}
