package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"
	"github.com/russross/blackfriday/v2"
	"github.com/winebarrel/cronplan"
)

const idxTmpl = `Show cron schedule.

USAGE:

  curl %s -d '5 0 * * ? *'

  curl %s -G --data-urlencode 'e=5 0 * * ? *'

see https://%s?e=5+0+*+*+?+*

cron expr spec: https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
implemented by https://github.com/winebarrel/cronplan
`

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
		host := c.Request.Host
		proto := c.GetHeader("X-Forwarded-Proto")
		ua := useragent.Parse(c.GetHeader("User-Agent"))

		if ua.Name != "curl" && proto != "https" && !strings.HasPrefix(host, "localhost") {
			if len(c.Request.URL.RawQuery) > 0 {
				host += "?" + c.Request.URL.RawQuery
			}

			c.Redirect(http.StatusFound, fmt.Sprintf("https://%s", host))
			return
		}

		exp, ok := c.GetQuery("e")

		if ok {
			schedule, err := cronNext(exp)

			if err != nil {
				c.String(http.StatusBadRequest, err.Error()+"\n")
				return
			}

			c.String(http.StatusOK, schedule)
		} else {
			index := fmt.Sprintf(idxTmpl, c.Request.Host, c.Request.Host, host)

			if ua.Name == "curl" {
				c.String(http.StatusOK, index)
			} else {
				c.Writer.Header().Set("Content-Type", "text/html")
				r := regexp.MustCompile(`(?m)^  `)
				index = r.ReplaceAllString(index, "&nbsp;&nbsp;")
				html := blackfriday.Run([]byte(index), blackfriday.WithNoExtensions(), blackfriday.WithExtensions(blackfriday.Autolink))
				c.String(http.StatusOK, string(html))
			}
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
