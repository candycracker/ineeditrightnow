package main

import (
	"fmt"
	"ineeditrightnow/src/search"
	"ineeditrightnow/src/store"
	"sync"
)

func main() {

	var wg sync.WaitGroup
	urls := search.GetAllRmtUrls(search.BuildGoogleMapSearchUrl())

	fmt.Println("-----------------------")
	jppUrls := []string{}
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			jppUrls = append(jppUrls, search.GetJaneappUrl(url)...)
		}(url)
	}
	wg.Wait()
	fmt.Println("-----------------------")
	cUrls := []string{}
	for _, url := range jppUrls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			cUrls = append(cUrls, search.GetCalendarUrls(url))
		}(url)
	}
	wg.Wait()

	for _, url := range cUrls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			search.GetCalendar(url)
		}(url)
	}
	wg.Wait()
	for a, b := range store.GetStore() {
		fmt.Printf("%s: %s,%d,%s,%s,%+v,%+v\n", a,
			b.URL,
			b.LocationID,
			b.Link,
			b.CalendarLink,
			b.Staffs,
			b.Disciplines)
	}

	// fmt.Println(search.GetAllRmtUrls(search.BuildGoogleMapSearchUrl()))
	// search.GetJaneappUrl("https://www.sensemassage.ca/")
	// search.GetJaneappUrl("http://www.momentumwellnesscentre.com/")
	// fmt.Println(search.GetCalendarUrls("https://sensemassage.janeapp.com/"))
	// fmt.Println("-----------------------")
}
