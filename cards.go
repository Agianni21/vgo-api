package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
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

// type definition to sort cards by rarity
type ByRarity Cards

func (a ByRarity) Len() int      { return len(a) }
func (a ByRarity) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRarity) Less(i, j int) bool {
	rarityI := rarityToInt(a[i].Rarity)
	rarityJ := rarityToInt(a[j].Rarity)
	return rarityI < rarityJ
}

type Cards []*Card

func RetrieveAllCards() Cards {
	// retrieves all the cards from the database
	var cards Cards
	db, err := sql.Open("sqlite3", "./virgio.db")
	defer db.Close()
	if err != nil {
		log.Print(err)
	}

	rows, err := db.Query("select * from Physicalcard inner join Card on Card.id = Physicalcard.card", nil)
	defer rows.Close()
	for rows.Next() {
		var card Card
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
		card.Owner = GetUserName(dbOwner)
		cards = append(cards, &card)
	}
	return cards
}

func (card Card) SaveCard() error {
	// for now it is only updating physical card info
	db, err := sql.Open("sqlite3", "./virgio.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := `update Physicalcard set contracts = ?, 
                                  inplay = ?, 
                                  owner = ?
                                  where id = ?`
	_, err = db.Exec(query,
		card.Contracts,
		BoolToInt(card.Inplay),
		GetUserId(card.Owner),
		card.Physicalid,
	)
	if err != nil {
		return err
	}
	return nil
}

func BoolToInt(value bool) int {
	if value {
		return 1
	} else {
		return 0
	}
}

func IntToBool(value int) bool {
	return value == 1
}
