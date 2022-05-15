package model

import "net/http"

type Resp struct {
	ResponseCode     int
	OldResponseCode  int
	ContentLength    int64
	OldContentLength int64
	ErrorCount       uint
}

func NewResp() *Resp {
	return &Resp{
		ResponseCode:     -1,
		OldResponseCode:  http.StatusOK,
		ContentLength:    -1,
		OldContentLength: -1,
		ErrorCount:       0,
	}
}
