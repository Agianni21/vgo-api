package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"fmt"
)

type Card struct {
	Physicalid int    `json:"physicalid"`
	Contracts  int    `json:"contracts"`
	Inplay     bool   `json:"inPlay"`
	Owner      string `json:"owner"`
	Cardid     int    `json:"id"`
	Name       string `json:"name"`
	Newcard    bool   `json:"new"`
	Notes      string `json:"notes"`
	Rarity     string `json:"rarity"`
	Rating     int    `json:"rating"`
	Cardtype   string `json:"type"`
	Picture    string `json:"picture"`
}

type ByRarity []Card

func (a ByRarity) Len() int      { return len(a) }
func (a ByRarity) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRarity) Less(i, j int) bool {
	rarityI := rarityToInt(a[i].Rarity)
	rarityJ := rarityToInt(a[j].Rarity)
	return rarityI < rarityJ
}

func listCardsAPI(w http.ResponseWriter, req *http.Request) {
	/*
		select cards
		insert into struct
		respond
	*/
	enableCors(&w)
	log.Print("GET /cards")
	db, err := sql.Open("sqlite3", "./virgio.db")
	if err != nil {
		log.Print(err)
	}
	rows, err := db.Query("select * from Physicalcard inner join Card on Card.id = Physicalcard.card", nil)
	var cards []Card
	var card Card
	for rows.Next() {
		card = Card{}
		var rawInplay int
		var rawNewcard int
		var dbOwner int
		var phycard int // useless value, but need to caputre it in row
		rows.Scan(
			&card.Physicalid,
			&card.Contracts,
			&rawInplay,
			&dbOwner,
			&phycard,
			&card.Cardid,
			&card.Name,
			&rawNewcard,
			&card.Notes,
			&card.Rarity,
			&card.Rating,
			&card.Cardtype,
		)
		card.Picture = fmt.Sprintf("https://storage.googleapis.com/ygoprodeck.com/pics/%d.jpg", card.Cardid)
		card.Inplay = rawInplay == 1
		card.Newcard = rawNewcard == 1
		card.Owner = getUserName(dbOwner)
		cards = append(cards, card)
	}
	responseBytes, err := json.Marshal(cards)
	if err != nil {
		log.Print(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

func buyPackAPI(w http.ResponseWriter, req *http.Request) {
	/*
		get user
		check funds
		if not enough, return some virgi garbage
		get cards
		change cards status
		return pack
	*/
	enableCors(&w)
	log.Print("GET /buypack")
	db, err := sql.Open("sqlite3", "./virgio.db")
	stmt, err := db.Prepare("select id, name, capital from User where name = ?")
	if err != nil {
		log.Print(err)
	}

	// Get user info
	username := req.URL.RawQuery

	rows, err := stmt.Query(username)
	if err != nil {
		log.Print(err)
	}
	log.Print(username)

	rows.Next()
	var userid int
	var name string
	var capital int
	rows.Scan(&userid, &name, &capital)

	rows.Close()
	stmt.Close()

	// Has enough virgo points
	if capital < 150 {
		w.WriteHeader(http.StatusPaymentRequired)
		return
	}

	// Get virgo cards
	rows, err = db.Query("select * from Physicalcard inner join Card on Card.id = Physicalcard.card", nil)
	if err != nil {
		log.Print(err)
	}
	cards := make(map[int]Card)
	var card Card
	for rows.Next() {
		card = Card{}
		var rawInplay int
		var rawNewcard int
		var dbOwner int
		var phycard int // useless value, but need to caputre it in row
		rows.Scan(
			&card.Physicalid,
			&card.Contracts,
			&rawInplay,
			&dbOwner,
			&phycard,
			&card.Cardid,
			&card.Name,
			&rawNewcard,
			&card.Notes,
			&card.Rarity,
			&card.Rating,
			&card.Cardtype,
		)
		card.Picture = fmt.Sprintf("https://storage.googleapis.com/ygoprodeck.com/pics/%d.jpg", card.Cardid)
		card.Inplay = rawInplay == 1
		card.Newcard = rawNewcard == 1
		card.Owner = getUserName(dbOwner)
		cards[card.Physicalid] = card
	}
	// Choose virgo pack cards
	rarities := packRarities()
	var packCards []Card
	for _, r := range rarities {
		card := getRandomCard(r, cards)
		card.Owner = getUserName(userid)
		cards[card.Physicalid] = card
		packCards = append(packCards, cards[card.Physicalid])
	}

	// Update virgo data
	newCapital := capital - 150
	_, err = db.Exec("update User set capital = ? where name = ?", newCapital, username)
	if err != nil {
		log.Print(err)
	}

	for _, card := range packCards {
		_, err = db.Exec("update Physicalcard set owner = ? where id = ?", userid, card.Physicalid)
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

func getRandomCard(rarity string, cards map[int]Card) Card {
	// Get all cards from virgo rarity
	var virgocards []Card
	for _, card := range cards {
		if card.Rarity == rarity && card.Owner == "none" && card.Inplay == true {
			virgocards = append(virgocards, card)
		}
	}
	// Choose virgo random
	chosen := rand.Intn(len(virgocards))
	return virgocards[chosen]
}

func getUserId(user string) int {
	switch user {
	case "Ale":
		return 1
	case "Bore":
		return 2
	case "Charly":
		return 3
	case "ChesterTester":
		return 0
	case "Juampi":
		return 5
	case "Maxi":
		return 6
	case "Nico":
		return 7
	case "Rodri":
		return 8
	case "Valen":
		return 4
	case "Nikito":
		return 9
	default:
		return 999
	}

}

func getUserName(user int) string {
	switch user {
	case 1:
		return "Ale"
	case 2:
		return "Bore"
	case 3:
		return "Charly"
	case 0:
		return "ChesterTester"
	case 5:
		return "Juampi"
	case 6:
		return "Maxi"
	case 7:
		return "Nico"
	case 8:
		return "Rodri"
	case 4:
		return "Valen"
	case 9:
		return "Nikito"
	default:
		return "none"
	}

}

func main() {
	log.Print("Starting server on port 8080")
	http.HandleFunc("/cards", listCardsAPI)
	http.HandleFunc("/buypack", buyPackAPI)
	http.ListenAndServe(":8080", nil)
}
