package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"site_monitoring/internal/model"
	"site_monitoring/internal/notification"
	"time"

	"github.com/sirupsen/logrus"
)

var m = make(map[string]*model.Resp)

type Client struct {
	C           http.Client
	Notificator notification.Notificator
	Logger      *logrus.Logger
}

func NewClient(log *logrus.Logger, notificator notification.Notificator) *Client {
	return &Client{
		C: http.Client{
			Timeout: 30 * time.Second,
		},
		Logger:      log,
		Notificator: notificator,
	}
}

func (c *Client) GetHeaders(ctx context.Context, url string) error {

	go func() {
		c.Logger.Infoln("Start checking ", url)
		ticker := time.NewTicker(5 * time.Second)

		resp := model.NewResp()

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			c.Logger.Fatalln("cannot create request")
		}

		for {
			select {
			case <-ctx.Done():
				c.Logger.Warnln("Timeout")
			case t := <-ticker.C:
				ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)

				req = req.WithContext(ctx)

				c.Logger.Infof("Checking %s at %s\n", url, t)

				res, err := c.C.Do(req)
				if err != nil {
					c.Logger.Errorln("cannot do request")
					resp.ResponseCode = 503
					resp.ContentLength = 0
				} else {
					resp.ResponseCode = res.StatusCode
					b, err := io.Copy(io.Discard, res.Body)
					if err != nil {
						cancel()
						c.Logger.Errorln("cannot get content-length")
						continue
					}
					err = res.Body.Close()
					if err != nil {
						cancel()
						c.Logger.Fatalln("cannot close resp.Body")
					}
					resp.ContentLength = b
				}

				m[url] = resp

				ok, err := CheckRespStatus(url)
				if err != nil {
					cancel()
					c.Logger.Errorln(err)
					continue
				}

				if !ok && resp.OldResponseCode != resp.ResponseCode {
					if res != nil {
						resp.OldResponseCode = res.StatusCode
					} else {
						resp.OldResponseCode = resp.ResponseCode
					}

					err = c.Notificator.SendMessage(fmt.Sprint("Server status not ", http.StatusOK, " in url: ", url,
						" at ", time.Now().Format("2006-01-02 15:04:05")))
					if err != nil {
						logrus.Errorln("cannot send tg message about server status")
					}
				} else if resp.OldResponseCode != resp.ResponseCode {
					resp.OldResponseCode = res.StatusCode
					err = c.Notificator.SendMessage(fmt.Sprint("Server started up in url: ", url,
						" at ", time.Now().Format("2006-01-02 15:04:05")))
					if err != nil {
						logrus.Errorln("cannot send tg message about server status")
					}
				} else if ok {
					ok, err = CheckRespContentLength(url)
					if err != nil {
						cancel()
						c.Logger.Errorln(err)
						continue
					}

					if !ok {
						err = c.Notificator.SendMessage(fmt.Sprint("Content-Length was changed in url: ", url,
							" at ", time.Now().Format("2006-01-02 15:04:05")))
						if err != nil {
							logrus.Errorln("cannot send tg message about content-length")
						}
					}
				}

				m[url] = resp

			}
		}

	}()

	return nil
}

func CheckRespStatus(url string) (bool, error) {
	if _, ok := m[url]; !ok {
		return false, fmt.Errorf("cannot get url: %s from map", url)
	}

	resp := m[url]

	return resp.ResponseCode == http.StatusOK, nil
}

func CheckRespContentLength(url string) (bool, error) {
	if _, ok := m[url]; !ok {
		return false, fmt.Errorf("cannot get url: %s from map", url)
	}

	ret := true
	resp := m[url]
	if resp.OldContentLength == -1 {
		resp.OldContentLength = resp.ContentLength
	}

	if resp.ContentLength != resp.OldContentLength {
		ret = false
		resp.OldContentLength = resp.ContentLength
	}

	m[url] = resp

	return ret, nil
}
