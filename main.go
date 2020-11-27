package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/caseymrm/menuet"
)

func temperature(woeid string) (temp, unit, text string) {
	requestUrl := "https://query.yahooapis.com/v1/public/yql?format=json&q=select%20item.condition%20from%20weather.forecast%20where%20woeid%20%3D%20" + woeid
	resp, err := http.Get(requestUrl)
	if err != nil {
		log.Fatal(err)
	}
	var response struct {
		Query struct {
			Results struct {
				Channel struct {
					Item struct {
						Condition struct {
							Temp string `json:"temp"`
							Text string `json:"text"`
						} `json:"condition"`
					} `json:"item"`
					Units struct {
						Temperature string `json:"temperature"`
					} `json:"units"`
				} `json:"channel"`
			} `json:"results"`
		} `json:"query"`
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&response)
	if err != nil {
		log.Fatal(err)
	}
	return response.Query.Results.Channel.Item.Condition.Temp, response.Query.Results.Channel.Units.Temperature, response.Query.Results.Channel.Item.Condition.Text
}

func location(query string) (string, string) {
	requestUrl := "https://query.yahooapis.com/v1/public/yql?format=json&q=select%20woeid,name%20from%20geo.places%20where%20text%3D%22" + url.QueryEscape(query) + "%22"
	resp, err := http.Get(requestUrl)
	if err != nil {
		log.Printf("Get: %v", err)
		menuet.App().Alert(menuet.Alert{
			MessageText:     "Could not get the weather",
			InformativeText: err.Error(),
		})
		return "", ""
	}
	var response struct {
		Query struct {
			Results struct {
				Place struct {
					Name  string `json:"name"`
					WoeID string `json:"woeid"`
				} `json:"place"`
			} `json:"results"`
		} `json:"query"`
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&response)
	if err != nil {
		log.Printf("Decode: %v", err)
		menuet.App().Alert(menuet.Alert{
			MessageText:     "Could not search for location",
			InformativeText: err.Error(),
		})
		return "", ""
	}
	return response.Query.Results.Place.Name, response.Query.Results.Place.WoeID
}

func temperatureString(woeid string) string {
	temp, unit, text := temperature(woeid)
	return fmt.Sprintf("%s°%s and %s", temp, unit, text)
}

func setWeather() {
	menuet.App().SetMenuState(&menuet.MenuState{
		Title: temperatureString(menuet.Defaults().String("loc")),
	})
}

var woeids = map[int]string{
	2442047: "Los Angeles",
	2487956: "San Francisco",
	2459115: "New York",
}

func menuPreview(woeid string) func() []menuet.MenuItem {
	return func() []menuet.MenuItem {
		return []menuet.MenuItem{
			menuet.MenuItem{
				Text: temperatureString(woeid),
				Clicked: func() {
					setLocation(woeid)
				},
			},
		}
	}
}

func menuItems() []menuet.MenuItem {
	items := []menuet.MenuItem{}

	currentWoeid := menuet.Defaults().String("loc")
	currentNumber, err := strconv.Atoi(currentWoeid)
	if err != nil {
		log.Printf("Atoi: %v", err)
	}
	found := false
	for woeid, name := range woeids {
		woeStr := strconv.Itoa(woeid)
		items = append(items, menuet.MenuItem{
			Text: name,
			Clicked: func() {
				setLocation(woeStr)
			},
			State:    woeStr == menuet.Defaults().String("loc"),
			Children: menuPreview(woeStr),
		})
		if woeid == currentNumber {
			found = true
		}
	}
	if !found {
		items = append(items, menuet.MenuItem{
			Text: menuet.Defaults().String("name"),
			Clicked: func() {
				setLocation(currentWoeid)
			},
			Children: menuPreview(currentWoeid),
			State:    true,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Text < items[j].Text
	})
	items = append(items, menuet.MenuItem{
		Text: "Other...",
		Clicked: func() {
			response := menuet.App().Alert(menuet.Alert{
				MessageText: "Where would you like to display the weather for?",
				Inputs:      []string{"Location"},
				Buttons:     []string{"Search", "Cancel"},
			})
			if response.Button == 0 && len(response.Inputs) == 1 && response.Inputs[0] != "" {
				newName, newWoeid := location(response.Inputs[0])
				if newWoeid != "" && newName != "" {
					menuet.Defaults().SetString("loc", newWoeid)
					menuet.Defaults().SetString("name", newName)
					menuet.App().Notification(menuet.Notification{
						Title:    fmt.Sprintf("Showing weather for %s", newName),
						Subtitle: temperatureString(newWoeid),
					})
					setWeather()
				}
			}
		},
	})
	return items
}

func hourlyWeather() {
	for {
		setWeather()
		time.Sleep(time.Hour)
	}
}

func setLocation(woeid string) {
	menuet.Defaults().SetString("loc", woeid)
	setWeather()
}

func main() {
	// Load the location from last time
	woeid := menuet.Defaults().String("loc")
	if woeid == "" {
		menuet.Defaults().SetString("loc", "2442047")
	}

	// Start the hourly check, and set the first value
	go hourlyWeather()

	// Configure the application
	menuet.App().Label = "com.github.caseymrm.menuet.weather"

	// Hook up the on-click to populate the menu
	menuet.App().Children = menuItems

	// Run the app (does not return)
	menuet.App().RunApplication()
}
