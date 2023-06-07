package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"ineeditrightnow/src/search"
	"ineeditrightnow/src/store"
	"sync"
	"time"
)

var treaments = map[string][]string{
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

func FindLocalStores(lat, lng string) []string {
	url := search.BuildGoogleMapSearchUrl(lat, lng)
	return search.GetAllRmtUrls(url)
}

func GetMerchant(url string) (store.Merchant, error) {

	if m := search.GetMerchant(url); m == nil {
		return store.Merchant{}, errors.New("unable to find Merchant Info")
	} else {
		return *m, nil
	}
}

// func Search(treatment string) string {

// }

func main() {

	var wg sync.WaitGroup
	urls := FindLocalStores("49.119423", "-123.1705926")
	fmt.Println(urls)

	fmt.Println("-----------------------")
	var merchants []store.Merchant
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if m, err := GetMerchant(url); err == nil {
				merchants = append(merchants, m)
				msg, _ := json.Marshal(m)
				fmt.Println(string(msg))
			}

		}(url)
	}
	wg.Wait()

	for _, m := range merchants {
		// for _, s := range m.Stores {
		// 	fmt.Println(m.Name, s.GetAllTreatments())
		// }
		m.Search("massage-therapy", time.Now())
	}

	// fmt.Println("-----------------------")
	// cUrls := []string{}
	// for _, url := range jppUrls {
	// 	wg.Add(1)
	// 	go func(url string) {
	// 		defer wg.Done()
	// 		cUrls = append(cUrls, search.GetCalendarUrls(url))
	// 	}(url)
	// }
	// wg.Wait()

	// for _, url := range cUrls {
	// 	wg.Add(1)
	// 	go func(url string) {
	// 		defer wg.Done()
	// 		search.GetCalendar(url)
	// 	}(url)
	// }
	// wg.Wait()
	// for a, b := range store.GetStore() {
	// 	fmt.Printf("%s: %s,%d,%s,%s,%+v,%+v\n", a,
	// 		b.URL,
	// 		b.LocationID,
	// 		b.Link,
	// 		b.CalendarLink,
	// 		b.Staffs,
	// 		b.Disciplines)
	// }

	// fmt.Println(search.GetAllRmtUrls(search.BuildGoogleMapSearchUrl()))
	// search.GetJaneappUrl("https://www.sensemassage.ca/")
	// search.GetJaneappUrl("http://www.momentumwellnesscentre.com/")
	// fmt.Println(search.GetCalendarUrls("https://sensemassage.janeapp.com/"))
	// fmt.Println("-----------------------")
}
