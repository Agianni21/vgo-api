package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sort"
)

func listCardsAPI(w http.ResponseWriter, req *http.Request) {
	enableCors(&w)
	log.Print("GET /cards")

	var cards Cards
	cards = RetrieveAllCards()
	// transform cards into json repsonse
	responseJson, err := json.Marshal(cards)
	if err != nil {
		log.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func getUserAPI(w http.ResponseWriter, req *http.Request) {
	enableCors(&w)
	log.Print("GET /user")

	// Get user
	username := req.URL.RawQuery
	user, err := GetUserByName(username)
	if err != nil {
		log.Print(err)
		return
	}

	// transform cards into json repsonse
	responseJson, err := json.Marshal(user)
	if err != nil {
		log.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func buyPackAPI(w http.ResponseWriter, req *http.Request) {
	enableCors(&w)
	log.Print("GET /buypack")

	// Get user
	username := req.URL.RawQuery
	user, err := GetUserByName(username)
	if err != nil {
		log.Print(err)
		return
	}

	// Has enough virgo points
	if user.Capital < 150 {
		w.WriteHeader(http.StatusPaymentRequired)
		return
	}

	// get all cards
	var cards Cards
	cards = RetrieveAllCards()

	// choose virgo cards of the pack
	rarities := packRarities()
	var packCards Cards
	for _, r := range rarities {
		card := getRandomCard(r, cards)
		card.Owner = user.Name
		packCards = append(packCards, card)
	}

	// update virgo user virgo points
	user.Capital = user.Capital - 150
	err = SaveUser(user)
	if err != nil {
		log.Print(err)
		return
	}

	// update virgo cards
	for _, card := range packCards {
		err = card.SaveCard()
		if err != nil {
			log.Print(err)
		}
	}

	// Create virgo response
	response := make(map[string]interface{})
	// sort by rarity
	sort.Sort(ByRarity(packCards))
	response["cards"] = packCards
	response["bestRarity"] = highestRarity(rarities)

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Print(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)

}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func highestRarity(rarities []string) string {
	var rarInts []int
	for _, rarity := range rarities {
		rarInts = append(rarInts, rarityToInt(rarity))
	}
	var highest int // virgo index
	for i, rarity := range rarInts {
		if rarInts[highest] < rarity {
			highest = i
		}
	}
	return rarities[highest]
}

func rarityToInt(rarity string) int {
	switch rarity {
	case "N":
		return 1
	case "R":
		return 2
	case "SR":
		return 3
	case "UR":
		return 4
	default:
		return 0
	}
}

func packRarities() []string {
	var rarities []string
	var firstTwo []string
	var last []string
	for i := 0; i < 60; i++ {
		firstTwo = append(firstTwo, "N")
		last = append(last, "R")
	}
	for i := 60; i < 90; i++ {
		firstTwo = append(firstTwo, "R")
		last = append(last, "SR")
	}
	for i := 90; i < 100; i++ {
		firstTwo = append(firstTwo, "SR")
		last = append(last, "UR")
	}
	rarities = append(rarities, firstTwo[rand.Intn(100)])
	rarities = append(rarities, firstTwo[rand.Intn(100)])
	rarities = append(rarities, last[rand.Intn(100)])
	return rarities
}

func getRandomCard(rarity string, cards Cards) *Card {
	// Get all cards from virgo rarity
	var virgocards Cards
	for _, card := range cards {
		if card.Rarity == rarity && card.Owner == "none" && card.Inplay == true {
			virgocards = append(virgocards, card)
		}
	}
	// Choose virgo random
	chosen := rand.Intn(len(virgocards))
	return virgocards[chosen]
}

func main() {
	log.Print("Starting server on port 8080")
	http.HandleFunc("/cards", listCardsAPI)
	http.HandleFunc("/buypack", buyPackAPI)
	http.HandleFunc("/user", getUserAPI)
	http.ListenAndServe(":8080", nil)
}
