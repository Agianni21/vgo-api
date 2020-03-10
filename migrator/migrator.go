package migrator

import (
	firestore "cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	//	"firebase.google.com/go/auth"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/api/option"
)

// Data types

type Card struct {
	Id       int
	Name     string
	NewCard  bool
	Notes    string
	Rarity   string
	Rating   int
	CardType string
}

type PhysicalCard struct { // puede cambiar, consultar con charly que son contratos, inplay
	Card      int
	Contracts int
	InPlay    bool
	Owner     string
}

type User struct {
	Name    string
	Capital int
	Salary  int
}

var (
	// hashmap where already read cards will be stored, the key is the card ID
	readCards         map[string]Card
	readPhysicalCards []PhysicalCard
	users             []User
)

func saveCollectionAsJson(collection string, toFile string) {
	ctx := context.Background()
	opt := option.WithCredentialsFile("key.json")

	client, err := firestore.NewClient(ctx, "ygo-ud", opt)
	must(err)

	rawCards := make(map[string]interface{})
	cards, err := client.Collection(collection).DocumentRefs(ctx).GetAll()
	must(err)

	for _, card := range cards {
		id := card.ID
		snapshot, err := card.Get(ctx)
		if err != nil {
			fmt.Errorf("error initializing app: %v", err)
		}
		data := snapshot.Data()
		rawCards[id] = data
	}

	b, err := json.Marshal(rawCards)
	must(err)

	err = ioutil.WriteFile(toFile, b, 0644)
	must(err)
}

func must(err error) {
	// lazy error checking
	if err != nil {
		log.Fatal(err)
	}
}

/*
after this function is called, the
	readCards
	readPhysicalCards
	user
variables should be filled with all the values extracted from
the json files
*/
func loadCardsFromJson() {
	cards := make(map[string]interface{})
	readCards = make(map[string]Card)
	// load cartas.json
	jsonString, err := ioutil.ReadFile("cartas.json")
	must(err)

	/* Unmarshal reads the json string and stores it
	in the map "cards" as a map of type [string]interface{},
	interface{} means that the value of the map can be anything
	(the keys of the map will be strings)
	*/
	err = json.Unmarshal(jsonString, &cards)
	must(err)

	// regex that will match and extract the id
	rgx := regexp.MustCompile(`(\d*)-?(\d*)`)

	for k, v := range cards {
		// the key 'k' correspond to the id

		// i have to cast v to the corresponding type of a "json object"
		jsoncard := v.(map[string]interface{})

		// extract id from possible id like 4438924-3 (no dash)
		str_cleanid := rgx.FindStringSubmatch(k)[1]
		// convert to int
		cleanid, err := strconv.Atoi(str_cleanid)
		must(err)

		// cast "new" to boolean
		newCardField := jsoncard["new"]
		var isNewCard = false
		if newCardField == "true" {
			isNewCard = true
		}

		// cast rating to int, if it fails (is string), assign to it 0
		// when reading a number from a json it always is a float64
		frating, ok := jsoncard["rating"].(float64)
		if !ok {
			frating = 0
		}
		rating := int(frating)

		// tengo que hacer esto porque una carta no tiene nombre (hay charli)
		cardname, ok := jsoncard["name"].(string)
		if !ok {
			cardname = "Yajiro Invader"
		}

		card := Card{
			Id:       cleanid,
			Name:     cardname,
			NewCard:  isNewCard,
			Notes:    jsoncard["notes"].(string),
			Rarity:   jsoncard["rarity"].(string),
			Rating:   rating,
			CardType: jsoncard["type"].(string),
		}

		// check if the card has already been read, if not store it
		_, isPresent := readCards[str_cleanid]
		if !isPresent {
			readCards[str_cleanid] = card
		}

		fcontracts, ok := jsoncard["contracts"].(float64)
		if !ok {
			fcontracts = 0
		}
		contracts := int(fcontracts)
		// create physical card
		pcard := PhysicalCard{
			Card:      card.Id,
			Contracts: contracts,
			InPlay:    jsoncard["inPlay"].(bool),
			Owner:     jsoncard["owner"].(string),
		}
		// add physical card to the rest
		readPhysicalCards = append(readPhysicalCards, pcard)
	}

	// Now we load the users

	// load usuarios.json
	jsonusers := make(map[string]interface{})
	jsonString, err = ioutil.ReadFile("usuarios.json")
	must(err)

	err = json.Unmarshal(jsonString, &jsonusers)
	must(err)
	for k, v := range jsonusers {
		jsonuser := v.(map[string]interface{})

		fsalary, ok := jsonuser["salary"].(float64)
		if !ok {
			fsalary = 0
		}
		salary := int(fsalary)

		fcapital, ok := jsonuser["capital"].(float64)
		if !ok {
			fcapital = 0
		}
		capital := int(fcapital)

		user := User{
			Name:    k,
			Salary:  salary,
			Capital: capital,
		}
		users = append(users, user)
	}
}

func fillCardTables() {
	// DB STUFF
	db, err := sql.Open("sqlite3", "./virgio.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	must(err)

	stmt, err := tx.Prepare(`insert into Card(id,
	                                          name,
						  newCard,
						  notes,
						  rarity,
						  rating,
						  type) 
				values (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range readCards {
		// sqlite doesn't have booleans
		var newCard int
		if v.NewCard {
			newCard = 1
		} else {
			newCard = 0
		}
		_, err = stmt.Exec(
			v.Id,
			v.Name,
			newCard,
			v.Notes,
			v.Rarity,
			v.Rating,
			v.CardType,
		)
		must(err)
	}
	tx.Commit()
	stmt.Close()

	// users
	tx, err = db.Begin()
	must(err)

	stmt, err = tx.Prepare(`insert into User(id,
	                                          name,
						  capital,
						  salary)
				values (?, ?, ?, ?)`)
	must(err)
	for _, user := range users {
		_, err = stmt.Exec(
			getUser(user.Name),
			user.Name,
			user.Capital,
			user.Salary,
		)
		must(err)
	}
	tx.Commit()
	stmt.Close()
	// physical cards
	tx, err = db.Begin()
	must(err)

	stmt, err = tx.Prepare(`insert into Physicalcard(contracts,
	                                          inplay,
						  owner,
						  card)
				values (?, ?, ?, ?)`)
	must(err)
	for _, pcard := range readPhysicalCards {
		var inplay int
		if pcard.InPlay {
			inplay = 1
		} else {
			inplay = 0
		}
		_, err = stmt.Exec(
			pcard.Contracts,
			inplay,
			getUser(pcard.Owner),
			pcard.Card,
		)
		must(err)
	}
	tx.Commit()
	stmt.Close()
}

func getUser(user string) int {
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
	default:
		return 999
	}

}

func createTables() {
	db, err := sql.Open("sqlite3", "virgio.db?_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Println("creating card table")
	createCard := `
        create table Card (
	    id integer not null primary key,
	    name text not null,
	    newCard integer not null,
	    notes text,
	    rarity text not null,
	    rating integer not null,
	    type string not null
	    );`

	_, err = db.Exec(createCard)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("creating user table")
	createUser := `
        create table User (
	    id integer primary key,
	    name text not null,
	    salary integer not null,
	    capital integer not null
	    );`

	_, err = db.Exec(createUser)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("creating phy card table")
	createPhyCard := `
        create table Physicalcard (
	    id integer primary key autoincrement,
	    contracts integer not null,
	    inplay integer not null,
	    owner integer not null,
	    card integer not null,
	    foreign key(owner)
	    references User(id)
	    	on update cascade
	    	on delete cascade,
	    foreign key(card)
	    references Card(id)
	    	on update cascade
	    	on delete cascade
	    );`

	_, err = db.Exec(createPhyCard)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	//	saveCollectionAsJson("usuarios", "usuarios.json")
	//	saveCollectionAsJson("descartas", "descartas.json")
	//	saveCollectionAsJson("cartas", "cartas.json")
	createTables()
	loadCardsFromJson()
	fillCardTables()
}
