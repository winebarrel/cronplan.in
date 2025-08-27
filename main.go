package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"
	"github.com/russross/blackfriday/v2"
	"github.com/winebarrel/cronplan/v2"
)

const idxTmpl = `Show AWS cron schedule.

USAGE:

  curl %[1]s -d '5 0 * * ? *'

  curl %[1]s/15 -d '*/5 10 ? * FRI *'

  curl -H 'accept: application/json' %[1]s -d '5 0 * * ? *'

  curl %[1]s -G --data-urlencode 'e=5 0 * * ? *'

  curl https://%[1]s/15?e=*/5+10+?+*+FRI+*

Cron expr spec: https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-cron-expressions.html

Implemented by https://github.com/winebarrel/cronplan
`

func cronNext(exp string, num string) ([]string, error) {
	cron, err := cronplan.Parse(exp)

	if err != nil {
		return nil, err
	}

	n := 10

	if num != "/" {
		num = strings.TrimPrefix(num, "/")
		fmt.Println(num)
		n, _ = strconv.Atoi(num)
	}

	triggers := cron.NextN(time.Now(), n)
	schedule := []string{}

	for _, t := range triggers {
		schedule = append(schedule, t.Format("Mon, 02 Jan 2006 15:04:05"))
	}

	return schedule, nil
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

			if c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusOK, map[string]any{
					"expr":     exp,
					"schedule": schedule,
				})
			} else {
				c.String(http.StatusOK, strings.Join(schedule, "\n")+"\n")
			}
		} else if num != "/" {
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
				c.String(http.StatusOK, fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>", "cronplan.in", string(html)))
			}
		}
	})

	r.POST("/*num", func(c *gin.Context) {
		num := c.Param("num")

		if !rPath.MatchString(num) {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		rawExp, err := io.ReadAll(c.Request.Body)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		exp := string(rawExp)
		schedule, err := cronNext(exp, num)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error()+"\n")
			return
		}

		if c.GetHeader("Accept") == "application/json" {
			c.JSON(http.StatusOK, map[string]any{
				"expr":     exp,
				"schedule": schedule,
			})
		} else {
			c.String(http.StatusOK, strings.Join(schedule, "\n")+"\n")
		}
	})

	addr := os.Getenv("LISTEN")
	port := os.Getenv("PORT")

	if addr == "" {
		addr = "127.0.0.1"
	}

	if port == "" {
		port = "8080"
	}

	err := r.Run(fmt.Sprintf("%s:%s", addr, port))

	if err != nil {
		log.Fatal(err)
	}
}
