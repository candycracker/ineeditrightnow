package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

var urlPrefix = "https://www.google.ca/maps/search/rmt+near+me/"
var lat string = "49.119423"
var lng string = "-123.1705926"
var rng string = "12z"
var treatment = "massage"
var duration = "60"

const RMT_PRICE_COEFFICIENT float64 = 1.8

var treamentPattern = map[string][]string{
	"massage": {"massage", "rmt"},
}

func BuildGoogleMapSearchUrl() string {
	return "https://www.google.ca/maps/search/rmt+near+me/@" + lat + "," + lng + "," + rng
}

type onHTML struct {
	selector string
	f        colly.HTMLCallback
}

func GetAllRmtUrls(url string) []string {

	rmtUrls := make(map[string]struct{})

	crawl(url, []onHTML{{"script", func(h *colly.HTMLElement) {
		regex := regexp.MustCompile(`http:\/\/www.\w+.com\/`)
		if urls := regex.FindAllString(h.Text, -1); len(urls) > 0 {
			for _, url := range urls {
				rmtUrls[url] = struct{}{}
			}
		}
	}}})

	urls := []string{}
	for url := range rmtUrls {
		urls = append(urls, url)
	}
	return urls
}

func GetJaneappUrl(url string) []string {

	jppUrls := make(map[string]struct{})

	crawl(url, []onHTML{{"a[href]", func(h *colly.HTMLElement) {
		val := h.Attr("href")
		regexUrl := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		regexUrlWithLocation := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/locations\/.*\/book`)
		url := regexUrl.FindString(val)
		urlWithLoc := regexUrlWithLocation.FindString(val)
		if url != "" {
			if urlWithLoc != "" {
				jppUrls[urlWithLoc] = struct{}{}
			} else {
				jppUrls[url] = struct{}{}
			}
		}
	}}, {"iframe", func(h *colly.HTMLElement) {
		val := h.Attr("src")
		regexUrl := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		url := regexUrl.FindString(val)
		if url != "" {
			jppUrls[url] = struct{}{}
		}
	}}})
	urls := []string{}
	for url := range jppUrls {
		urls = append(urls, url)
	}
	return urls
}

func convertUrl(url string, locID, discID, trmtID int) string {
	newUrl := fmt.Sprintf("%sapi/v2/openings/for_discipline?location_id=%d&discipline_id=%d&treatment_id=%d&date=&num_days=7", url, locID, discID, trmtID)
	return newUrl
}

func GetCalendarUrls(url string) string {

	cldUrls := ""

	crawl(url, []onHTML{{"a[href]", func(h *colly.HTMLElement) {

		treamentAttr := h.DOM.Find("strong").Text()

		if len(treamentAttr) == 0 {
			return
		}

		treamentAttr = strings.ToLower(treamentAttr)

		priceAttr := h.DOM.Find("small").Text()

		regTreaments := []*regexp.Regexp{}
		for _, v := range treamentPattern[treatment] {
			regTreaments = append(regTreaments, regexp.MustCompile(v))
		}
		found := false
		for _, regex := range regTreaments {
			found = found || regex.MatchString(treamentAttr)
		}
		regDuration := regexp.MustCompile(duration)
		if !regDuration.MatchString(treamentAttr) || !found {
			return
		}
		regPrice := regexp.MustCompile(`\$\d*.\d*`)

		priceStr := regPrice.FindString(priceAttr)

		if len(priceStr) == 0 {
			return
		}

		price, err := strconv.ParseFloat(priceStr[1:], 64)
		dur, _ := strconv.Atoi(duration)
		if err != nil {
			return
		}
		if price < float64(dur)*RMT_PRICE_COEFFICIENT {
			return
		}

		p := strings.Split(h.Attr("href"), "/")
		locationID := 1
		disciplineID, _ := strconv.Atoi(p[2])
		treatmentID, _ := strconv.Atoi(p[4])
		cldUrls = convertUrl(url, locationID, disciplineID, treatmentID)

	}}})
	return cldUrls
}

func crawl(url string, on []onHTML) {

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting: %s\n", r.URL)
	})

	for _, o := range on {
		c.OnHTML(o.selector, o.f)
	}

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("Error: %s\n", e.Error())
	})
	c.Visit(url)
}

type Calendar struct {
	StaffMemberID       int    `json:"staff_member_id"`
	LocationID          int    `json:"location_id"`
	TreatmentID         int    `json:"treatment_id"`
	Duration            int    `json:"duration"`
	StartAt             string `json:"start_at"`
	EndAt               string `json:"end_at"`
	RoomID              int    `json:"room_id"`
	CallToBook          bool   `json:"call_to_book"`
	State               string `json:"state"`
	Status              string `json:"status"`
	ParentAppointmentID string `json:"parent_appointment_id"`
}

func GetCalendar(url string) []Calendar {

	data := []Calendar{}
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data
	}
	req.Header.Add("Cookie", "_front_desk_session=")
	resp, err := client.Do(req)
	if err != nil {
		return data
	}

	defer resp.Body.Close()

	if resp.StatusCode > 400 {
		fmt.Println("Status code: ", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return data
	}

	json.Unmarshal(body, &data)

	return data
}
