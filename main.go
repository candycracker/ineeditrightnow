package main

import (
	"fmt"
	"ineeditrightnow/src/search"
	"sync"
)

const API_KEY string = "AIzaSyDyYN_n-M5HE_1BOrJOEoUzD9Bg-n3eUCE"

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
			fmt.Printf("%+v\n", search.GetCalendar(url))
		}(url)
	}
	wg.Wait()

	fmt.Println("-----------------------")
}
