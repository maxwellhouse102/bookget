package app

import (
	"bookget/config"
	"bookget/lib/gohttp"
	"bookget/lib/util"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
)

type Tnm struct {
	dt *DownloadTask
}

func (r Tnm) Init(iTask int, sUrl string) (msg string, err error) {
	r.dt = new(DownloadTask)
	r.dt.UrlParsed, err = url.Parse(sUrl)
	r.dt.Url = sUrl
	r.dt.Index = iTask
	r.dt.BookId = r.getBookId(r.dt.Url)
	if r.dt.BookId == "" {
		return "requested URL was not found.", err
	}
	r.dt.Jar, _ = cookiejar.New(nil)
	return r.download()
}

func (r Tnm) getBookId(sUrl string) (bookId string) {
	m := regexp.MustCompile(`(?i)/dlib/detail/([A-z0-9_-]+)`).FindStringSubmatch(sUrl)
	if m != nil {
		bookId = m[1]
	}
	return bookId
}

func (r Tnm) download() (msg string, err error) {
	name := util.GenNumberSorted(r.dt.Index)
	log.Printf("Get %s  %s\n", name, r.dt.Url)

	r.dt.SavePath = config.CreateDirectory(r.dt.Url, r.dt.BookId)
	apiUrl := fmt.Sprintf("%s://%s/dlib/pages/%s", r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, r.dt.BookId)
	canvases, err := r.getCanvases(apiUrl, r.dt.Jar)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.Printf(" %d pages \n", len(canvases))
	return r.do(canvases)
}

func (r Tnm) do(dziUrls []string) (msg string, err error) {
	if dziUrls == nil {
		return "", err
	}
	referer := url.QueryEscape(r.dt.Url)
	args := []string{
		"-H", "Origin:" + referer,
		"-H", "Referer:" + referer,
		"-H", "User-Agent:" + config.Conf.UserAgent,
	}
	for i, uri := range dziUrls {
		if config.SeqContinue(i) {
			continue
		}
		if uri == "" {
			continue
		}
		sortId := util.GenNumberSorted(i + 1)
		log.Printf("Get %s  %s\n", sortId, uri)
		filename := sortId + config.Conf.FileExt
		dest := r.dt.SavePath + string(os.PathSeparator) + filename
		util.StartProcess(uri, dest, args)
	}
	return "", err
}

func (r Tnm) getVolumes(sUrl string, jar *cookiejar.Jar) (volumes []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r Tnm) getCanvases(sUrl string, jar *cookiejar.Jar) (canvases []string, err error) {
	bs, err := r.getBody(sUrl, jar)
	if err != nil {
		return
	}
	type ResponseBody struct {
		ImageType string `json:"imageType"`
		Imageid   string `json:"imageid"`
		Ready     bool   `json:"ready"`
		Id        int    `json:"id"`
		Path      string `json:"path"`
	}
	var result []ResponseBody
	if err = json.Unmarshal(bs, &result); err != nil {
		return
	}
	for _, v := range result {
		xmlUrl := fmt.Sprintf("%s://%s/dlib/img/%s/tiles/%s/ImageProperties.xml",
			r.dt.UrlParsed.Scheme, r.dt.UrlParsed.Host, r.dt.BookId, v.Imageid)
		canvases = append(canvases, xmlUrl)
	}
	return canvases, nil
}

func (r Tnm) getBody(apiUrl string, jar *cookiejar.Jar) ([]byte, error) {
	referer := url.QueryEscape(apiUrl)
	cli := gohttp.NewClient(gohttp.Options{
		CookieFile: config.Conf.CookieFile,
		CookieJar:  jar,
		Headers: map[string]interface{}{
			"User-Agent": config.Conf.UserAgent,
			"Referer":    referer,
		},
	})
	resp, err := cli.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	bs, _ := resp.GetBody()
	if resp.GetStatusCode() == 202 || bs == nil {
		return nil, errors.New(fmt.Sprintf("ErrCode:%d, %s", resp.GetStatusCode(), resp.GetReasonPhrase()))
	}
	return bs, nil
}
