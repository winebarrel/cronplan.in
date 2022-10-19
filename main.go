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

func cronNext(exp string) (string, error) {
	cron, err := cronplan.Parse(exp)

	if err != nil {
		return "", err
	}

	triggers := cron.NextN(time.Now(), 10)
	var buf strings.Builder

	for _, t := range triggers {
		buf.WriteString(t.Format("Mon, 02 Jan 2006 15:04:05"))
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		exp, ok := c.GetQuery("e")

		if ok {
			schedule, err := cronNext(exp)

			if err != nil {
				c.String(http.StatusBadRequest, err.Error()+"\n")
				return
			}

			c.String(http.StatusOK, schedule)
		} else {
			c.String(http.StatusOK, fmt.Sprintf(`Show cron schedule.

USAGE:

  curl %s -d '5 0 * * ? *'

  curl %s -G --data-urlencode 'e=5 0 * * ? *'

see http://%s?e=5+0+*+*+?+*

implemented by https://github.com/winebarrel/cronplan
`, c.Request.Host, c.Request.Host, c.Request.Host))
		}
	})

	r.POST("/", func(c *gin.Context) {
		exp, err := io.ReadAll(c.Request.Body)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		schedule, err := cronNext(string(exp))

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		c.String(http.StatusOK, schedule)
	})

	r.Run()
}
