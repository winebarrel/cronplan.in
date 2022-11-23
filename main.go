package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"
	"github.com/russross/blackfriday/v2"
	"github.com/winebarrel/cronplan"
)

const idxTmpl = `Show AWS cron schedule.

USAGE:

  curl %[1]s -d '5 0 * * ? *'

  curl %[1]s/15 -d '*/5 10 ? * FRI *'

  curl %[1]s -G --data-urlencode 'e=5 0 * * ? *'

  e.g. https://%[1]s/15?e=*/5+10+?+*+FRI+*

cron expr spec: https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html#CronExpressions

implemented by https://github.com/winebarrel/cronplan
`

func cronNext(exp string, num string) (string, error) {
	cron, err := cronplan.Parse(exp)

	if err != nil {
		return "", err
	}

	n := 10

	if num != "/" {
		num = strings.TrimPrefix(num, "/")
		fmt.Println(num)
		n, _ = strconv.Atoi(num)
	}

	triggers := cron.NextN(time.Now(), n)
	var buf strings.Builder

	for _, t := range triggers {
		buf.WriteString(t.Format("Mon, 02 Jan 2006 15:04:05"))
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

func main() {
	r := gin.Default()
	rPath := regexp.MustCompile(`^/\d*$`)

	r.GET("/*num", func(c *gin.Context) {
		num := c.Param("num")

		if !rPath.MatchString(num) {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

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
			schedule, err := cronNext(exp, num)

			if err != nil {
				c.String(http.StatusBadRequest, err.Error()+"\n")
				return
			}

			c.String(http.StatusOK, schedule)
		} else if num != "/" {
			fmt.Println(num)
			c.Redirect(http.StatusFound, "/")
		} else {
			index := fmt.Sprintf(idxTmpl, host)

			if ua.Name == "curl" {
				c.String(http.StatusOK, index)
			} else {
				c.Writer.Header().Set("Content-Type", "text/html")
				r := regexp.MustCompile(`(?m)^  `)
				index = r.ReplaceAllString(index, "&nbsp;&nbsp;")
				html := blackfriday.Run([]byte(index), blackfriday.WithNoExtensions(), blackfriday.WithExtensions(blackfriday.Autolink))
				c.String(http.StatusOK, fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>", "cronplan.io", string(html)))
			}
		}
	})

	r.POST("/*num", func(c *gin.Context) {
		num := c.Param("num")

		if !rPath.MatchString(num) {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		exp, err := io.ReadAll(c.Request.Body)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		schedule, err := cronNext(string(exp), num)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		c.String(http.StatusOK, schedule)
	})

	addr := os.Getenv("LISTEN")
	port := os.Getenv("PORT")

	if addr == "" {
		addr = "127.0.0.1"
	}

	if port == "" {
		port = "8080"
	}

	r.Run(fmt.Sprintf("%s:%s", addr, port))
}
