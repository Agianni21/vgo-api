package main

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Id      int
	Name    string
	Capital int
	Salary  int
}

func GetUserByName(name string) (User, error) {
	var user User
	db, err := sql.Open("sqlite3", "./virgio.db")
	if err != nil {
		return user, err
	}
	defer db.Close()

	rows, err := db.Query("select id, name, capital, salary from User where name = ?", name)
	if err != nil {
		return user, err
	}
	defer rows.Close()
	// assume that only one user will be returned from query
	if !rows.Next() {
		return user, errors.New("no user with that name")
	}
	rows.Scan(&user.Id, &user.Name, &user.Capital, &user.Salary)
	return user, nil
}

func SaveUser(user User) error {
	db, err := sql.Open("sqlite3", "./virgio.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := `update User set capital = ?, salary = ? where id = ?`
	_, err = db.Exec(query, user.Capital, user.Salary, user.Id)
	if err != nil {
		return err
	}
	return nil
}

func GetUserId(user string) int {
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

func GetUserName(user int) string {
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
