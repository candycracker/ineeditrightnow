package search

import (
	"fmt"
	"ineeditrightnow/src/store"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

var urlPrefix = "https://www.google.ca/maps/search/rmt+near+me/@"
var rng string = "12z"
var reqTreatment = "massage"
var reqDuration = 60
var regexJaneappUrl = regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
var regexUrlWithLocation = regexp.MustCompile(`https:\/\/(.*/?).janeapp.com\/locations\/(.*/?)\/book`)
var regexMerchant = regexp.MustCompile(`https://([a-zA-Z\d]+).janeapp.com/`)
var regexAddress = regexp.MustCompile(`at:!(.*/?)!!Dir`)
var regexPhone = regexp.MustCompile(`\d+[.-]\d+[.-]\d+`)
var regexEmail = regexp.MustCompile(`([a-zA-Z|\d]+)@(.*?)\.com`)
var regexDisciplineAndTreatment = regexp.MustCompile(`#/discipline/\d+/treatment/\d+`)
var regexLocation = regexp.MustCompile(`App.location_id = (\d+)`)

const RMT_PRICE_COEFFICIENT float64 = 1.8

var treamentPattern = map[string][]string{
	"massage":          {"registered massage therapy", "rmt", "massage therapy", "massage"},
	"osteopathy":       {"osteopathy"},
	"acupuncture":      {"acu", "acupuncture"},
	"bodywork":         {"bodywork"},
	"chiropractic":     {"chiropractic"},
	"pilates":          {"pilates"},
	"athletic therapy": {"athletic therapy"},
	"counsell":         {"counsell"},
	"vestibular":       {"vestibular"},
	"physiotherapy":    {"physio", "physiotherapy"},
}

var regTreaments = make(map[string][]*regexp.Regexp)
var regPrice = regexp.MustCompile(`\$\d*.\d*`)
var durRegex = regexp.MustCompile(`\d+\s?min`)

func init() {
	for treament, patterns := range treamentPattern {
		regex := []*regexp.Regexp{}
		for _, p := range patterns {
			regex = append(regex, regexp.MustCompile(p))
		}
		regTreaments[treament] = regex
	}
}

func BuildGoogleMapSearchUrl(lat, lng string) string {
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

func getJaneappUrl(orgUrl string) string {

	jUrl := ""
	crawl(orgUrl, []onHTML{{"a[href]", func(h *colly.HTMLElement) {
		val := h.Attr("href")
		if url := regexJaneappUrl.FindString(val); url != "" {
			jUrl = url
		}
	}}, {"iframe", func(h *colly.HTMLElement) {
		val := h.Attr("src")
		if url := regexJaneappUrl.FindString(val); url != "" {
			jUrl = url
		}
	}}})

	return jUrl
}

func GetMerchant(orgUrl string) *store.Merchant {

	jUrl := getJaneappUrl(orgUrl)

	var m = new(store.Merchant)

	if finds := regexMerchant.FindStringSubmatch(jUrl); len(finds) == 2 {
		m = &store.Merchant{Name: finds[1], Website: jUrl, Stores: []*store.Store{}}
		store.AddNewMerchant(finds[1], m)
	} else {
		return nil
	}

	multiLoc := false

	crawl(jUrl, []onHTML{{"a[href]", func(h *colly.HTMLElement) {

		val := h.Attr("href")
		if url := regexUrlWithLocation.FindString(val); url != "" {
			if finds := regexUrlWithLocation.FindStringSubmatch(val); len(finds) == 3 {
				subvision := finds[2]
				s := store.NewStore(subvision, url)
				m.Stores = append(m.Stores, s)
			}
			multiLoc = true
		}

	}}})
	if !multiLoc {
		s := store.NewStore("#", jUrl)
		m.Stores = append(m.Stores, s)
	}

	for _, store := range m.Stores {

		crawl(store.Url, []onHTML{{"div.col-sm-offset-3", func(h *colly.HTMLElement) {

			x := strings.Replace(h.Text, "\n", "!", -1)
			phone := regexPhone.FindString(x)
			email := regexEmail.FindString(x)
			address := ""

			if finds := regexAddress.FindStringSubmatch(x); len(finds) == 2 {
				address = finds[1]
			}
			store.Address = address
			store.Phone = phone
			store.Email = email

		}}, {
			"script", func(h *colly.HTMLElement) {

				if strings.Contains(h.Text, "location_id") {
					find := regexLocation.FindString(h.Text)
					idStr := regexLocation.FindStringSubmatch(find)[1]
					id, _ := strconv.Atoi(idStr)
					store.LocationID = id
				}

			},
		}})

		GetCalendarUrls(store)

	}

	return m
}

func getTreamentAttr(attr string) string {
	for t, regexs := range regTreaments {
		for _, regex := range regexs {
			if regex.MatchString(attr) {
				return t
			}
		}
	}
	return ""
}

func getDurationAttr(attr string) int {

	if durRegex.MatchString(attr) {
		durStr := regexp.MustCompile(`\d+`).FindString(durRegex.FindString(attr))
		if d, err := strconv.Atoi(durStr); err == nil {
			return d
		}
	}

	return -1
}

func getTreamentAndDuration(attr string) (string, int) {

	treatment := ""
	for t, regexs := range regTreaments {
		for _, regex := range regexs {
			if regex.MatchString(attr) {
				treatment = t
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

func GetCalendarUrls(store *store.Store) string {
	cldUrls := ""
	crawl(store.Url, []onHTML{{"div.discipline-container", func(h *colly.HTMLElement) {
		h.ForEach("section", func(_ int, e *colly.HTMLElement) {

			content := ""
			for _, node := range e.DOM.Nodes {
				if len(node.Attr) != 1 {
					//TO DO: not sure if there is other case that have multiple attr
					continue
				}
				content = node.Attr[0].Val
				//TO DO: not sure if there is other case that have multiple e.DOM.Nodes
				break
			}
			if content == "" {
				return
			}

			e.ForEach("nav", func(_ int, nav *colly.HTMLElement) {
				navigation := nav.Attr("aria-labelledby")

				raw := strings.Split(navigation, "_")
				if len(raw) != 4 {
					return
				}
				id, err := strconv.Atoi(raw[1])
				if err != nil {
					return
				}

				discipline := store.GetDiscipline(content)

				if discipline == nil {
					discipline = store.AddNewDiscipline(content, id, content)
				}

				if strings.HasSuffix(navigation, "staff_navigation") {
					nav.ForEach("a[href].photo", func(_ int, s *colly.HTMLElement) {
						staffID := strings.Split(s.Attr("href"), "/")[2]
						staff := s.ChildText("div.hidden-xs")
						store.AddStaff(staff, staffID)
						store.Staffs[staffID].AddNewTreatment(id)
					})
				} else if strings.HasSuffix(navigation, "treatments_navigation") {
					nav.ForEach("a[href]", func(_ int, s *colly.HTMLElement) {

						attr := s.Attr("href")
						if !regexDisciplineAndTreatment.MatchString(attr) {
							return
						}
						content1 := s.ChildText("strong")
						content2 := s.ChildText("small")
						content1 = strings.ToLower(content1)
						content2 = strings.Replace(content2, "\n", " ", -1)
						content2 = strings.ToLower(content2)

						duration := getDurationAttr(content1)
						if duration == -1 {
							duration = getDurationAttr(content2)
						}
						price := getPrice(content2)

						if price == -1 {
							price = getPrice(content1)
						}

						hrefContent := strings.Split(attr, "/")
						treatmentID, _ := strconv.Atoi(hrefContent[4])

						discipline.AddNewTreatment(treatmentID, duration, price, content1)
					})
				}

			})

		})

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
