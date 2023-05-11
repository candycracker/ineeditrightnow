package search

import (
	"encoding/json"
	"fmt"
	"ineeditrightnow/src/store"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

var urlPrefix = "https://www.google.ca/maps/search/rmt+near+me/@"
var lat string = "49.119423"
var lng string = "-123.1705926"
var rng string = "12z"
var reqTreatment = "massage"
var reqDuration = 60

const RMT_PRICE_COEFFICIENT float64 = 1.8

var treamentPattern = map[string][]string{
	"massage":      {"registered massage therapy", "rmt"},
	"osteopathy":   {"osteopathy"},
	"acupuncture":  {"acupuncture"},
	"bodywork":     {"bodywork"},
	"chiropractic": {"chiropractic"},
}

var regTreaments = make(map[string][]*regexp.Regexp)
var regPrice = regexp.MustCompile(`\$\d*.\d*`)
var durRegex = regexp.MustCompile(`\d*.?min`)

func init() {
	for treament, patterns := range treamentPattern {
		regex := []*regexp.Regexp{}
		for _, p := range patterns {
			regex = append(regex, regexp.MustCompile(p))
		}
		regTreaments[treament] = regex
	}
}

func BuildGoogleMapSearchUrl() string {
	return urlPrefix + lat + "," + lng + "," + rng
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

func GetJaneappUrl(link string) []string {

	locationID := 1
	crawl(link, []onHTML{{"a[href]", func(h *colly.HTMLElement) {

		val := h.Attr("href")
		regexUrl := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		regexUrlWithLocation := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/locations\/.*\/book`)

		if regexUrlWithLocation.MatchString(val) {
			url := regexUrlWithLocation.FindString(val)
			if store.FindStore(url) == nil {
				store.AddStore(url, locationID, link)
				locationID++
			}
		} else if regexUrl.MatchString(val) {
			url := regexUrl.FindString(val)
			if store.FindStore(url) == nil {
				store.AddStore(url, locationID, link)
			}
		} else {
			return
		}
	}}, {"iframe", func(h *colly.HTMLElement) {
		val := h.Attr("src")
		regexUrl := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		if regexUrl.MatchString(val) {
			url := regexUrl.FindString(val)
			if store.FindStore(url) == nil {
				store.AddStore(url, locationID, link)
			}
		}
	}}})

	urls := []string{}
	for _, s := range store.GetStore() {
		urls = append(urls, s.URL)
	}
	return urls
}

func convertUrl(url string, locID, discID, trmtID int) string {
	newUrl := fmt.Sprintf("%sapi/v2/openings/for_discipline?location_id=%d&discipline_id=%d&treatment_id=%d&date=&num_days=7", url, locID, discID, trmtID)
	return newUrl
}

func getTreamentAndDuration(attr string) (string, int) {

	treatment := ""
	for t, regexs := range regTreaments {
		for _, regex := range regexs {
			if regex.MatchString(attr) {
				treatment = t
				if regexp.MustCompile(`icbc`).MatchString(attr) {
					treatment = "icbc " + treatment
				}
				break
			}
		}
	}
	durStr := ""

	if durRegex.MatchString(attr) {
		dur := durRegex.FindString(attr)
		durStr = regexp.MustCompile(`\d*`).FindString(dur)
	}
	d, err := strconv.Atoi(durStr)
	if err != nil {
		return "", -1
	}

	return treatment, d
}

func getPrice(attr string) float64 {

	priceStr := regPrice.FindString(attr)

	if len(priceStr) == 0 {
		return -1
	}

	price, err := strconv.ParseFloat(priceStr[1:], 64)
	if err != nil {
		return -1
	}
	return price
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

		treament, duration := getTreamentAndDuration(treamentAttr)
		price := getPrice(priceAttr)

		if duration == -1 || price == -1 {
			return
		}

		// if reqTreatment != treament && reqDuration != duration {
		// 	return
		// }
		// if price < float64(duration)*RMT_PRICE_COEFFICIENT {
		// 	return
		// }

		p := strings.Split(h.Attr("href"), "/")
		fmt.Println(url, p, treament, duration)
		store := store.FindStore(url)
		if store == nil {
			return
		}

		locationID := store.LocationID
		disciplineID, _ := strconv.Atoi(p[2])
		treatmentID, _ := strconv.Atoi(p[4])
		store.AddDisciplines(disciplineID, treament)
		store.Disciplines[disciplineID].AddTreatment(treatmentID, duration, price)
		// fmt.Println(">>>", locationID, disciplineID, treatmentID, treament, duration, price)
		cldUrls = convertUrl(url, locationID, disciplineID, treatmentID)
		store.CalendarLink = cldUrls
	}}, {
		"a[href].photo", func(h *colly.HTMLElement) {
			staffID := strings.Split(h.Attr("href"), "/")[2]
			staff := h.ChildText("div.hidden-xs")
			store := store.FindStore(url)
			if store == nil {
				return
			}
			store.AddStaff(staff, staffID)
		},
	}})
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
