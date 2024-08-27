package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"text/template"
	"time"

	"linguatron/models"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func startMessage() string {
	return "Starting app..."
}

func main() {
	fmt.Println(startMessage())

	var err error
	db, err = sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		panic(err)
	}

	createCardsTable(db)
	createDecksTable(db)

	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/create-deck", CreateDeckHandler)
	http.HandleFunc("/decks", DecksHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	fmt.Println("Server starting at :8080")
	http.ListenAndServe(":8080", nil)

}
func HomeHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("Welcome to Linguatron!"))
}

func CreateDeckHandler(writer http.ResponseWriter, request *http.Request) {
	displayForm := func() {
		tmpl, _ := template.ParseFiles("./templates/create_deck.html")
		data := struct {
			Title   string
			Heading string
			Message string
		}{
			Title:   "Deck Creation",
			Heading: "Create a deck",
			Message: "All you need to create a deck is a deck name. Duplicate deck names are allowed.",
		}
		tmpl.Execute(writer, data)
	}
	processForm := func() {
		err := request.ParseForm()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		deckName := request.FormValue("deckname")
		insertDeck(db, deckName)

		fmt.Fprintf(writer, "<div id='result'>Deck '%s' created successfully!</div>", deckName)

	}

	switch request.Method {
	case "GET":
		displayForm()
	case "POST":
		processForm()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}

func DecksHandler(writer http.ResponseWriter, request *http.Request) {
	displayDecks := func() {
		tmpl, _ := template.ParseFiles("./templates/decks.html")
		decks, err := selectAllDecks(db)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Title string
			Decks []models.Deck
		}{
			Title: "List of Decks",
			Decks: decks,
		}
		tmpl.Execute(writer, data)
	}

	switch request.Method {
	case "GET":
		displayDecks()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

// possible values from stage: learning, review
func createCardsTable(db *sql.DB) error {
	_, err := db.Exec(`
                CREATE TABLE IF NOT EXISTS cards (
                        card_id INTEGER PRIMARY KEY,
                        deck_id INTEGER,
                        correct INTEGER DEFAULT 0,
                        incorrect INTEGER DEFAULT 0,
                        card_created TEXT DEFAULT "",
                        last_review_date TEXT DEFAULT "",
                        stage TEXT DEFAULT "learning",
                        lapses INTEGER DEFAULT 0,
                        ease INTEGER DEFAULT 1,
                        review_due_date TEXT DEFAULT "",
                        question TEXT,
                        answer TEXT,
                        FOREIGN KEY (deck_id) REFERENCES decks(deck_id) ON DELETE CASCADE
                )
        `)
	return err
}

func createDecksTable(db *sql.DB) error {
	_, err := db.Exec(`
                CREATE TABLE IF NOT EXISTS decks (
                        deck_id INTEGER PRIMARY KEY,
                        deck_name TEXT
                )
        `)
	return err
}

func selectAllDecks(db *sql.DB) ([]models.Deck, error) {
	rows, err := db.Query("SELECT * FROM decks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decks []models.Deck
	for rows.Next() {
		var deck models.Deck
		if err := rows.Scan(&deck.DeckID, &deck.Deckname); err != nil {
			return nil, err
		}
		decks = append(decks, deck)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return decks, nil
}

func selectDeckByID(db *sql.DB, deckID int) (models.Deck, error) {
	var deck models.Deck
	err := db.QueryRow("SELECT deck_id, deck_name FROM decks WHERE deck_id = ?", deckID).Scan(&deck.DeckID, &deck.Deckname)
	if err != nil {
		return models.Deck{}, err
	}
	return deck, nil
}

func insertDeck(db *sql.DB, deckName string) error {
	_, err := db.Exec("INSERT INTO decks (deck_name) VALUES (?)", deckName)
	return err
}

func deleteDeck(db *sql.DB, deckID int) error {
	_, err := db.Exec("DELETE FROM decks WHERE deck_id = ?", deckID)
	return err
}

func insertCard(db *sql.DB, deckID int, question, answer, cardCreated string) error {
	_, err := db.Exec("INSERT INTO cards (deck_id, question, answer, card_created) VALUES (?, ?, ?, ?)", deckID, question, answer, cardCreated)
	return err
}

func deleteCard(db *sql.DB, cardID int) error {
	_, err := db.Exec("DELETE FROM cards WHERE card_id = ?", cardID)
	return err
}

func selectCardByCardID(db *sql.DB, cardID int) (models.Card, error) {

	var card models.Card
	if err := db.QueryRow("SELECT * FROM cards WHERE card_id = ?", cardID).Scan(&card.CardID, &card.DeckID, &card.Correct, &card.Incorrect, &card.CardCreated, &card.LastReviewDate, &card.Stage, &card.Lapses, &card.Ease, &card.ReviewDueDate, &card.Question, &card.Answer); err != nil {
		return models.Card{}, err
	}
	return card, nil
}

func selectLearningCardsByDeckID(db *sql.DB, deckID int) ([]models.Card, error) {
	rows, err := db.Query("SELECT * FROM cards WHERE deck_id = ? AND stage = 'learning'", deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var card models.Card
		if err := rows.Scan(&card.CardID, &card.DeckID, &card.Correct, &card.Incorrect, &card.CardCreated, &card.LastReviewDate, &card.Stage, &card.Lapses, &card.Ease, &card.ReviewDueDate, &card.Question, &card.Answer); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func selectReviewCardsByDeckID(db *sql.DB, deckID int) ([]models.Card, error) {
	rows, err := db.Query("SELECT * FROM cards WHERE deck_id = ? AND stage = 'review'", deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var card models.Card
		if err := rows.Scan(&card.CardID, &card.DeckID, &card.Correct, &card.Incorrect, &card.CardCreated, &card.LastReviewDate, &card.Stage, &card.Lapses, &card.Ease, &card.ReviewDueDate, &card.Question, &card.Answer); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func selectAllCardsByDeckID(db *sql.DB, deckID int) ([]models.Card, error) {
	rows, err := db.Query("SELECT * FROM cards WHERE deck_id = ?", deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var card models.Card
		if err := rows.Scan(&card.CardID, &card.DeckID, &card.Correct, &card.Incorrect, &card.CardCreated, &card.LastReviewDate, &card.Stage, &card.Lapses, &card.Ease, &card.ReviewDueDate, &card.Question, &card.Answer); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func updateLearningCard(db *sql.DB, cardID int, correctAnswer bool) error {
	stmt, err := db.Prepare("UPDATE cards SET correct = ?, incorrect = ?, ease = ?, stage = ?, review_due_date = ?, last_review_date = ? WHERE card_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	card, err := selectCardByCardID(db, cardID)
	if err != nil {
		return err
	}
	//TODO add stage logic
	if correctAnswer {
		_, err = stmt.Exec(card.Correct+1, card.Incorrect, getNextEaseLevel(card.Ease, 2), card.Stage, createNextReviewDueDate(getNextEaseLevel(card.Ease, 2)), time.Now().UTC().String(), cardID)
	} else {
		_, err = stmt.Exec(card.Correct, card.Incorrect+1, 1, card.Stage, time.Now().UTC().String(), time.Now().UTC().String(), cardID)
	}

	return err
}

func updateReviewCard(db *sql.DB, cardID int, correctAnswer bool) error {
	stmt, err := db.Prepare("UPDATE cards SET correct = ?, incorrect = ?, lapses = ?, ease = ?, review_due_date = ?, last_review_date = ? WHERE card_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	card, err := selectCardByCardID(db, cardID)
	if err != nil {
		return err
	}
	if correctAnswer {
		_, err = stmt.Exec(card.Correct+1, card.Incorrect, card.Lapses, getNextEaseLevel(card.Ease, 2), createNextReviewDueDate(getNextEaseLevel(card.Ease, 2)), time.Now().UTC().String(), cardID)
	} else {
		//increment lapses by 1 if ease is not 1 AND answer is incorrect
		if card.Ease != 1 {
			_, err = stmt.Exec(card.Correct, card.Incorrect+1, 1, card.Lapses+1, time.Now().UTC().String(), time.Now().UTC().String(), cardID)
		} else {
			_, err = stmt.Exec(card.Correct, card.Incorrect+1, 1, card.Lapses, time.Now().UTC().String(), time.Now().UTC().String(), cardID)
		}
	}

	return err
}

func getNextEaseLevel(currentEase int, growthfactor float64) int {
	growthFactor := growthfactor
	nextEase := int(math.Ceil(float64(currentEase) * growthFactor))

	return nextEase
}

func createNextReviewDueDate(ease int) string {
	t := time.Now().UTC()
	return t.Add(time.Duration(ease) * 24 * time.Hour).String()
}
