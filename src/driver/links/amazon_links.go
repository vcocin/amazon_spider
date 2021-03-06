package driverlink

import (
	"crypto/md5"
	"curl"
	"fmt"
	"net/url"
	"ssdb"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/golang/glog"
)

func GetProductLinks(znum int, wg *sync.WaitGroup) {
	ssdbtool.SSDBPool.SetLinkQueue()
	for index := 0; index < znum; index++ {
		go start(wg)
	}
}

func start(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		targetUrl, tailKey, pageLog, err := ssdbtool.SSDBPool.GetQueueLink()
		if err != nil {
			glog.Warningf("get target links error  : %+v", err)
			break
		}
		if targetUrl == "" {
			size, err := ssdbtool.SSDBPool.GetQueueSize()
			if err != nil {
				glog.Errorf("get size error")
			}
			if size == 0 {
				glog.Warningln("tail queue empty")
				break
			}
		}

		rdata, err := curl.GetURLData(targetUrl)
		if err != nil {
			glog.Errorf("Curl Error : %+v", err)
		}

		time.Sleep(time.Microsecond * 500)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(rdata))
		if err != nil {
			glog.Errorf("Parser links : %v\n   Error : %+v", targetUrl, err)
			continue
		}

		totalPage, err := strconv.ParseInt(strings.Trim(doc.Find(".pagnDisabled").First().Text(), " "), 10, 32)
		if err != nil {
			glog.Warningf("get page links : %v\n   Error : %+v", targetUrl, err)
			continue
		}

		glog.Infof("target : %v | totalpage : %v", targetUrl, pageLog)

		for sp := pageLog; sp <= int(totalPage); sp++ {
			pdata := make(map[string]interface{})
			target := fmt.Sprintf("%v&page=%v", targetUrl, sp)
			rdata, err := curl.GetURLData(target)
			if err != nil {
				glog.Errorf("Curl Error : %+v", err)
				continue
			}

			time.Sleep(time.Microsecond * 500)

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(rdata))
			if err != nil {
				glog.Errorf("Parser links : %v\n   Error : %+v", targetUrl, err)
				continue
			}

			root := doc.Find(".s-item-container")
			if root.Size() == 0 {
				glog.Warningf("Nil Product Page : %s", target)
				break
			}
			root.Each(func(i int, s *goquery.Selection) {
				result, ok := s.Find(".s-access-detail-page").First().Attr("href")
				if ok {
					res, err := url.Parse(result)
					if err != nil {
						glog.Warningf("Url Parse Error : %+v", err)
					}
					pid := fmt.Sprintf("%x", md5.Sum([]byte(res.EscapedPath())))
					productUrl := fmt.Sprintf("https://%v%v\n\n", res.Hostname(), res.EscapedPath())
					pdata[pid] = productUrl
				}
			})
			ssdbtool.SSDBPool.SetProductLink(pdata)
			ssdbtool.SSDBPool.SavePageLog(tailKey, sp)
		}
	}
}
