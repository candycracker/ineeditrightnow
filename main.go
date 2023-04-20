package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

/*
																											https://richmondremedymassage.janeapp.com/locations/remedy-massage-therapy/book#/discipline/1/treatment/1
Visiting: http://www.momentumwellnesscentre.com/			https://momentumwellnesscentre.janeapp.com/		https://momentumwellnesscentre.janeapp.com/locations/momentum-steveston-village/book#/discipline/2/treatment/5
Visiting: http://www.satoriintegrativehealth.com/			https://satori.janeapp.com/						https://satori.janeapp.com/#/discipline/2/treatment/7
Visiting: http://www.raintreespa.com/
Visiting: http://www.evolvetherapeutic.com/					https://evolvetherapy.janeapp.com/				https://evolvetherapy.janeapp.com/#/discipline/1/treatment/3
Visiting: http://www.truephysiopilates.com/					https://truephysiopilates.janeapp.com/			https://truephysiopilates.janeapp.com/#/discipline/2/treatment/8
Visiting: http://www.lifelogicintegrativehealth.com/
Visiting: http://www.taodayspa.com/
Visiting: http://www.massageempathy.com/					https://massageempathy.janeapp.com/				https://massageempathy.janeapp.com/#/discipline/1/treatment/23
*/

// var url string = "https://richmondremedymassage.janeapp.com/api/v2/openings/for_discipline?location_id=1&discipline_id=1&treatment_id=1&date=&num_days=7"
var urlsMap = make(map[string]struct{})
var janeappUrlsMap = make(map[string]struct{})
var calendarUrlsMap = make(map[string]Params)
var Lat string = "49.192991"
var Lng string = "-123.173890"
var Rng string = "10z"
var Discipline string = "massage"
var Duration string = "60"

type Params struct {
	LocationID   int
	DisciplineID int
	TreatmentID  int
}

const RMT_PRICE_COEFFICIENT float64 = 1.8

const (
	MASSAGE       = 1
	PHYSIOTHERAPY = 2
)

func main() {

	var wg sync.WaitGroup
	crawlRmtNearby()
	for url := range urlsMap {
		wg.Add(1)
		url := url
		go func() {
			defer wg.Done()
			crawl(url)
		}()
	}
	wg.Wait()
	fmt.Println("-----------------------")
	for url := range janeappUrlsMap {
		crawl2(url, Discipline, Duration)
	}
	for url := range calendarUrlsMap {
		goQuery(convertUrl(url))
		// fmt.Println()
	}
	// goQuery("https://richmondremedymassage.janeapp.com/api/v2/openings/for_discipline?location_id=1&discipline_id=1&treatment_id=1&date=&num_days=7")
	// goQuery("https://truephysiopilates.janeapp.com/#/discipline/2/treatment/8")
	//          https://truephysiopilates.janeapp.com/api/v2/openings/for_discipline?location_id=1&discipline_id=2&treatment_id=8&date=&num_days=7
}

func convertUrl(url string) string {
	newUrl := fmt.Sprintf("%sapi/v2/openings/for_discipline?location_id=%d&discipline_id=%d&treatment_id=%d&date=&num_days=7", url, calendarUrlsMap[url].LocationID, calendarUrlsMap[url].DisciplineID, calendarUrlsMap[url].TreatmentID)
	// fmt.Println(newUrl)
	return newUrl
}

func check(e error) {
	if e != nil {
		fmt.Println(e)
	}
}

func goQuery(url string) {

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Cookie", "_front_desk_session=")

	client := &http.Client{}
	resp, err := client.Do(req)

	check(err)
	defer resp.Body.Close()

	if resp.StatusCode > 400 {
		fmt.Println("Status code: ", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	check(err)
	html := doc.Selection.Text()
	check(err)
	fmt.Printf("%+v\n", html)
}

func writeFile(data, name string) {
	file, err := os.Create(name)
	check(err)
	defer file.Close()

	file.WriteString(data)
}

func crawl2(url string, treatment string, duration string) {

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting janeapp: %s\n", r.URL)
	})

	// Find and print all links
	c.OnHTML("a[href]", func(h *colly.HTMLElement) {
		//find all url
		// fmt.Println(">> ", h)
		//
		// val := h.Attr("strong")
		// a := h.Attr("a")
		strong := h.DOM.Find("strong").Text()
		if strong != "" {
			strong = strings.ToLower(strong)
			small := h.DOM.Find("small").Text()
			regStrong := regexp.MustCompile(duration + `.*` + treatment)
			regSmall := regexp.MustCompile(`\$\d*.\d*`)
			if regStrong.MatchString(strong) {
				price := 0.0
				priceStr := regSmall.FindString(small)

				// fmt.Println(h.Attr("href"))
				if len(priceStr) > 0 {
					if p, err := strconv.ParseFloat(priceStr[1:], 64); err == nil {
						price = p
					} else {
						fmt.Printf("%v\n", err)
					}
				}
				dur, _ := strconv.Atoi(duration)
				if price > float64(dur)*RMT_PRICE_COEFFICIENT {
					p := strings.Split(h.Attr("href"), "/")
					locationID := 1
					disciplineID, _ := strconv.Atoi(p[2])
					treatmentID, _ := strconv.Atoi(p[4])
					// fmt.Println(">>>", h.Attr("href"))
					calendarUrlsMap[url] = Params{LocationID: locationID, DisciplineID: disciplineID, TreatmentID: treatmentID}
					// fmt.Println(regSmall.FindString(small))
					// fmt.Println("----------------")
				}
			}
		}

		// fmt.Printf("%+v\n", h)
		// reg := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		// url := reg.FindString(val)
		// if url != "" {
		// 	janeappUrlsMap[url] = struct{}{}
		// }
	})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("Error: %s\n", e.Error())
	})
	c.Visit(url)
}

func crawl(url string) {

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting: %s\n", r.URL)
	})

	// Find and print all links
	c.OnHTML("a[href]", func(h *colly.HTMLElement) {
		//find all url
		val := h.Attr("href")
		reg := regexp.MustCompile(`https:\/\/\w+.janeapp.com\/`)
		url := reg.FindString(val)
		if url != "" {
			janeappUrlsMap[url] = struct{}{}
		}
	})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("Error: %s\n", e.Error())
	})
	c.Visit(url)
}

func crawlRmtNearby() {
	c := colly.NewCollector(
		colly.AllowedDomains("https://www.google.ca", "www.google.ca"),
	)

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting: %s\n", r.URL)
	})
	// Find and print all links
	c.OnHTML("script", func(h *colly.HTMLElement) {
		//find all rmt links
		reg := regexp.MustCompile(`http:\/\/www.\w+.com\/`)
		urls := reg.FindAllString(h.Text, -1)
		if len(urls) != 0 {
			for _, url := range urls {
				urlsMap[url] = struct{}{}
			}
			fmt.Printf("%d: %v\n", len(urls), urls)
		}
	})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("Error: %s\n", e.Error())
	})

	c.Visit("https://www.google.ca/maps/search/rmt+near+me/@" + Lat + "," + Lng + "," + Rng)
}
