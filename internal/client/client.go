package client

import (
	"context"
	"fmt"
	"net/http"
	"site_monitoring/internal/model"
	"site_monitoring/internal/notification"
	"site_monitoring/sitemon/config"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var m = make(map[string]*model.Resp)

type Client struct {
	sync.RWMutex
	C           http.Client
	Notificator notification.Notificator
	Logger      *logrus.Logger
	Cfg         *config.Config
}

func NewClient(log *logrus.Logger, notificator notification.Notificator, cfg *config.Config) *Client {
	return &Client{
		C: http.Client{
			Timeout: 30 * time.Second,
		},
		Logger:      log,
		Notificator: notificator,
		Cfg:         cfg,
	}
}

func (c *Client) GetHeaders(ctx context.Context, url string, wg *sync.WaitGroup) error {

	go func() {
		defer func() {
			wg.Done()
		}()

		c.Logger.Infoln("Start checking ", url)
		ticker := time.NewTicker(c.Cfg.Timing.Delay)

		resp := model.NewResp()

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			c.Logger.Fatalln("cannot create request")
		}

		ctx2, cancel2 := context.WithTimeout(ctx, 30*time.Second)
		t := time.Now()
		c.Do(ctx2, cancel2, url, resp, req, t)

		for {
			select {
			case <-ctx.Done():
				c.Logger.Warnln("exiting...")
				return
			case t = <-ticker.C:
				ctx2, cancel2 = context.WithTimeout(ctx, 30*time.Second)
				c.Do(ctx2, cancel2, url, resp, req, t)
			}
		}

	}()

	return nil
}

func (c *Client) CheckRespStatus(url string) (bool, error) {
	c.RLock()
	if _, ok := m[url]; !ok {
		return false, fmt.Errorf("cannot get url: %s from map", url)
	}

	resp := m[url]
	c.RUnlock()

	return resp.ResponseCode == http.StatusOK, nil
}

func (c *Client) CheckRespContentLength(url string) (bool, error) {
	c.RLock()
	if _, ok := m[url]; !ok {
		return false, fmt.Errorf("cannot get url: %s from map", url)
	}
	resp := m[url]
	c.RUnlock()

	ret := true

	if resp.OldContentLength == -1 {
		resp.OldContentLength = resp.ContentLength
	}

	if resp.ContentLength != resp.OldContentLength {
		ret = false
		resp.OldContentLength = resp.ContentLength
	}

	c.Lock()
	m[url] = resp
	c.Unlock()

	return ret, nil
}

func (c *Client) Do(ctx context.Context, cancel context.CancelFunc, url string, resp *model.Resp, req *http.Request, t time.Time) {
	req = req.WithContext(ctx)

	c.Logger.Infof("Checking %s at %s\n", url, t)

	res, err := c.C.Do(req)
	if err != nil {
		c.Logger.Errorln("cannot do request: ", err.Error())
		if strings.Contains(err.Error(), ": context canceled") {
			cancel()
			return
		}
		resp.ResponseCode = 503
		resp.ContentLength = 0
	} else {
		resp.ResponseCode = res.StatusCode
		/*b, err := io.Copy(io.Discard, res.Body)
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
		resp.ContentLength = b*/
	}

	c.Lock()
	m[url] = resp
	c.Unlock()

	ok, err := c.CheckRespStatus(url)
	if err != nil {
		cancel()
		c.Logger.Errorln(err)
		return
	}

	if !ok && (resp.OldResponseCode != resp.ResponseCode || resp.ErrorCount > 0) {
		if res != nil {
			resp.OldResponseCode = res.StatusCode
		} else {
			resp.OldResponseCode = resp.ResponseCode
		}
		resp.ErrorCount++
		logrus.Warnln(resp.ErrorCount)

		if resp.ErrorCount == 3 {
			err = c.Notificator.SendMessage(fmt.Sprint("Server down. Status ", resp.ResponseCode, " in url: ", url,
				" at ", time.Now().Format("2006-01-02 15:04:05")))
			if err != nil {
				logrus.Errorln("cannot send tg message about server status")
			}
		}

	} else if resp.OldResponseCode != resp.ResponseCode {
		resp.OldResponseCode = res.StatusCode
		resp.ErrorCount = 0
		err = c.Notificator.SendMessage(fmt.Sprint("Server started up in url: ", url,
			" at ", time.Now().Format("2006-01-02 15:04:05")))
		if err != nil {
			logrus.Errorln("cannot send tg message about server status")
		}
	} /*else if ok {
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
	}*/

	c.Lock()
	m[url] = resp
	c.Unlock()
}
