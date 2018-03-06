package extractors

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/iawia002/annie/config"
	"github.com/iawia002/annie/downloader"
	"github.com/iawia002/annie/request"
	"github.com/iawia002/annie/utils"
)

const (
	// BiliBili blocks keys from time to time.
	// You can extract from the Android client or bilibiliPlayer.min.js
	appKey string = "84956560bc028eb7"
	secKey string = "94aba54af9065f71de72f5508f1cd42e"
)

type dURLData struct {
	Size  int64  `json:"size"`
	URL   string `json:"url"`
	Order int    `json:"order"`
}

type bilibiliData struct {
	DURL    []dURLData `json:"durl"`
	Format  string     `json:"format"`
	Quality int        `json:"quality"`
}

// {"code":0,"message":"0","ttl":1,"data":{"token":"aaa"}}
// {"code":-101,"message":"账号未登录","ttl":1}
type tokenData struct {
	Token string `json:"token"`
}

type token struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    tokenData `json:"data"`
}

func getSign(params string) string {
	sign := md5.New()
	sign.Write([]byte(params + secKey))
	return fmt.Sprintf("%x", sign.Sum(nil))
}

func genAPI(aid, cid string, bangumi bool) string {
	var (
		baseAPIURL string
		params     string
	)
	if bangumi {
		// The parameters need to be sorted by name
		// qn=0 flag makes the CDN address different every time
		// quality=116(1080P 60) is the highest quality so far
		params = fmt.Sprintf(
			"appkey=%s&cid=%s&module=bangumi&otype=json&qn=116&quality=116&season_type=4&type=",
			appKey, cid,
		)
		baseAPIURL = config.BILIBILI_BANGUMI_API
	} else {
		params = fmt.Sprintf(
			"appkey=%s&cid=%s&otype=json&qn=116&quality=116&type=",
			appKey, cid,
		)
		baseAPIURL = config.BILIBILI_API
	}
	utoken := ""
	if config.Cookie != "" {
		utoken = request.Get(fmt.Sprintf(
			"%said=%s&cid=%s", config.BILIBILI_TOKEN_API, aid, cid,
		))
		var t token
		json.Unmarshal([]byte(utoken), &t)
		if t.Code != 0 {
			log.Println(config.Cookie)
			log.Fatal("Cookie error: ", t.Message)
		}
		utoken = t.Data.Token
	}
	api := fmt.Sprintf(
		"%s%s&sign=%s&utoken=%s", baseAPIURL, params, getSign(params), utoken,
	)
	return api
}

func genURL(durl []dURLData) ([]downloader.URLData, int64) {
	var (
		urls []downloader.URLData
		size int64
	)
	for _, data := range durl {
		size += data.Size
		url := downloader.URLData{
			URL:  data.URL,
			Size: data.Size,
			Ext:  "flv",
		}
		urls = append(urls, url)
	}
	return urls, size
}

// Bilibili download function
func Bilibili(url string) downloader.VideoData {
	var (
		bangumi bool
		cid     string
	)
	if strings.Contains(url, "bangumi") {
		bangumi = true
	}
	aid := utils.Match1(`av(\d+)`, url)[1]
	html := request.Get(url)
	if bangumi {
		cid = utils.Match1(`"cid":(\d+)`, html)[1]
	} else {
		cid = utils.Match1(`cid=(\d+)`, html)[1]
	}
	api := genAPI(aid, cid, bangumi)
	apiData := request.Get(api)
	var dataDict bilibiliData
	json.Unmarshal([]byte(apiData), &dataDict)

	// get the title
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Fatal(err)
	}
	var title string
	title = strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "" {
		// Some movie page got no h1 tag
		title, _ = doc.Find("meta[property=\"og:title\"]").Attr("content")
	}

	urls, size := genURL(dataDict.DURL)
	data := downloader.VideoData{
		Site:  "哔哩哔哩 bilibili.com",
		Title: utils.FileName(title),
		URLs:  urls,
		Type:  "video",
		Size:  size,
	}
	data.Download(url)
	return data
}
