package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/rand"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Deck struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Cards []Card `gorm:"foreignKey:DeckID"`
}

type Card struct {
	ID             uint `gorm:"primaryKey"`
	DeckID         uint
	Correct        uint   `gorm:"default:0"`
	Incorrect      uint   `gorm:"default:0"`
	CardCreated    string `gorm:"default:''"`
	LastReviewDate string `gorm:"default:''"`
	Stage          string `gorm:"default:'learning'"`
	Lapses         uint   `gorm:"default:0"`
	Ease           uint   `gorm:"default:1"`
	ReviewDueDate  string `gorm:"default:''"`
	Question       string
	Answer         string
}

type Database interface {
	createDeck(name string) error
	createCard(card Card) error
	getCardByID(id uint) (Card, error)
	getAllCardsByDeckID(id uint) ([]Card, error)
	getRandomCardsByDeckID(id uint) ([]Card, error)
	getLearningCardsByDeckID(id uint) ([]Card, error)
	getReviewCardsByDeckID(id uint) ([]Card, error)
	getDueReviewCardsByDeckID(id uint) ([]Card, error)
	getDeckByID(id uint) (Deck, error)
	selectAllDecks() ([]Deck, error)
	updateLearningCardByID(card Card) error
	updateReviewCardByID(card Card) error
	deleteCardByID(card Card) error
}

type GormDB struct {
	db *gorm.DB
}

func (g *GormDB) createDeck(name string) error {
	var deck Deck
	deck.Name = name
	return g.db.Create(&deck).Error
}

func (g *GormDB) createCard(card Card) error {
	return g.db.Create(&card).Error
}

func (g *GormDB) getCardByID(id uint) (Card, error) {
	var card Card
	err := g.db.First(&card, id).Error
	return card, err
}

func (g *GormDB) getAllCardsByDeckID(id uint) ([]Card, error) {
	var cards []Card
	err := g.db.Where("deck_id = ?", id).Find(&cards).Error
	return cards, err
}

func (g *GormDB) getRandomCardsByDeckID(deckID uint, cardID uint) ([]Card, error) {
	var count int64
	cardCountError := g.db.Model(&Card{}).Where("deck_id = ?", deckID).Count(&count).Error
	if cardCountError != nil {
		return nil, cardCountError
	}

	var limit int

	if count >= 3 && count < 5 {
		limit = 3
	} else if count >= 5 {
		limit = 5
	} else {
		limit = 0
	}

	var cards []Card
	err := g.db.Where("deck_id = ? AND id != ?", deckID, cardID).Order("RANDOM()").Limit(limit).Find(&cards).Error

	return cards, err
}

func (g *GormDB) getLearningCardsByDeckID(id uint) ([]Card, error) {
	var cards []Card
	err := g.db.Where("deck_id = ? AND stage = ?", id, "learning").Find(&cards).Error
	return cards, err
}
func (g *GormDB) getReviewCardsByDeckID(id uint) ([]Card, error) {
	var cards []Card
	err := g.db.Where("deck_id = ? AND stage = ?", id, "review").Find(&cards).Error
	return cards, err
}
func (g *GormDB) getDueReviewCardsByDeckID(id uint) ([]Card, error) {
	now := time.Now().UTC()

	var cards []Card
	err := g.db.Where("deck_id = ? AND stage = ? AND review_due_date <= ?", id, "review", now).Find(&cards).Error
	return cards, err
}

func (g *GormDB) getDeckByID(id uint) (Deck, error) {
	var deck Deck
	err := g.db.First(&deck, id).Error
	return deck, err
}

func (g *GormDB) selectAllDecks() ([]Deck, error) {
	var decks []Deck
	err := g.db.Find(&decks).Error
	return decks, err
}

func (g *GormDB) updateLearningCardByID(id uint, correct bool) error {
	card, _ := g.getCardByID(id)

	now, _ := time.Now().UTC().MarshalText()

	minuteAfter, _ := time.Now().UTC().Add(time.Minute * time.Duration(1)).MarshalText()

	dayAfter, _ := time.Now().UTC().Add(time.Hour * time.Duration(24)).MarshalText()

	card.LastReviewDate = string(now)
	if correct {
		card.Correct++
		if card.Ease > 1 {
			card.Ease = uint(getNextEaseLevel(int(card.Ease), 1))
			card.Stage = "review"
			card.ReviewDueDate = string(dayAfter)

		} else {
			card.Ease = uint(getNextEaseLevel(int(card.Ease), 2))
			card.ReviewDueDate = string(minuteAfter)
		}
	} else {
		card.Incorrect++
		card.Ease = 1
		card.ReviewDueDate = string(minuteAfter)
	}
	return g.db.Save(&card).Error
}

func (g *GormDB) updateReviewCardByID(id uint, correct bool) error {
	card, _ := g.getCardByID(id)

	now, _ := time.Now().UTC().MarshalText()

	minuteAfter, _ := time.Now().UTC().Add(time.Minute * time.Duration(1)).MarshalText()

	card.LastReviewDate = string(now)
	if correct {
		card.Correct++
		card.Ease = uint(getNextEaseLevel(int(card.Ease), 2))
		card.ReviewDueDate = createNextReviewDueDate(getNextEaseLevel(int(card.Ease), 2))
	} else {
		card.Incorrect++
		card.ReviewDueDate = string(minuteAfter)
		if card.Ease != 1 {
			card.Lapses++
			card.Ease = 1
		}
	}

	return g.db.Save(&card).Error
}

func startMessage() string {
	return "Starting app..."
}

func HomeHandler(writer http.ResponseWriter, request *http.Request) {

	displayHome := func() {
		tmpl, _ := template.ParseFiles("./templates/index.html", "./templates/navbar.html")
		tmpl.Execute(writer, nil)
	}

	switch request.Method {
	case "GET":
		displayHome()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func (g *GormDB) CreateDeckHandler(writer http.ResponseWriter, request *http.Request) {
	displayForm := func() {
		tmpl, _ := template.ParseFiles("./templates/create_deck.html", "./templates/navbar.html")
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
		g.createDeck(deckName)

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

func IsAnswerCorrectInLowerCase(userAnswer string, databaseAnswer string) bool {
	return strings.ToLower(userAnswer) == strings.ToLower(databaseAnswer)
}
func getMostDueCard(cards []Card) (Card, error) {
	// timeLayout := "2024-08-30T12:57:22.141705535Z"
	var mostDueCard Card
	if len(cards) > 0 {
		mostDueCard = cards[0]
	}

	if len(cards) > 1 {
		mostDueCardTime, err := time.Parse(time.RFC3339Nano, cards[0].ReviewDueDate)
		if err != nil {
			fmt.Print(err.Error())
			return Card{}, err
		}

		for i := 0; i < len(cards); i++ {
			currentCardDueDate, err := time.Parse(time.RFC3339Nano, cards[i].ReviewDueDate)
			if err != nil {
				fmt.Print(err.Error())
				return Card{}, err
			}
			if currentCardDueDate.Before(mostDueCardTime) {
				mostDueCardTime = currentCardDueDate
				mostDueCard = cards[i]
			}

		}
		return mostDueCard, err
	} else if len(cards) == 1 {
		return mostDueCard, nil
	} else {
		return Card{}, fmt.Errorf("no cards")
	}

}

func (g *GormDB) DeckHandler(writer http.ResponseWriter, request *http.Request) {
	IDString := strings.TrimPrefix(request.URL.Path, "/deck/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))
	cards, _ := g.getAllCardsByDeckID(uint(id))

	cardsJSON, _ := json.Marshal(cards)

	displayCards := func() {
		tmpl, _ := template.ParseFiles("./templates/deck.html", "./templates/navbar.html")
		data := struct {
			Title string
			Deck  Deck
			Cards template.JS
		}{
			Title: "Deck " + deck.Name,
			Deck:  deck,
			Cards: template.JS(cardsJSON),
		}
		tmpl.Execute(writer, data)
	}

	switch request.Method {
	case "GET":
		displayCards()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}

func (g *GormDB) LearningMultipleChoiceHandler(writer http.ResponseWriter, request *http.Request) {

	IDString := strings.TrimPrefix(request.URL.Path, "/learning-multiple-choice/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))

	displayLearning := func() {
		cards, _ := g.getLearningCardsByDeckID(deck.ID)
		mostDueCard, _ := getMostDueCard(cards)
		var cardAvailable bool

		randomCards, _ := g.getRandomCardsByDeckID(deck.ID, mostDueCard.ID)
		randomCards = append(randomCards, mostDueCard)
		rand.Shuffle(len(randomCards), func(i, j int) {
			randomCards[i], randomCards[j] = randomCards[j], randomCards[i]
		})

		if len(randomCards) > 3 {
			cardAvailable = true
		} else {
			cardAvailable = false
		}
		tmpl, _ := template.ParseFiles("./templates/htmx/learning-multiple-choice.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			RandomCards   []Card
			MostDueCard   Card
			CardAvailable bool
		}{
			Title:         "Deck: " + deck.Name,
			Deck:          deck,
			RandomCards:   randomCards,
			MostDueCard:   mostDueCard,
			CardAvailable: cardAvailable,
		}

		tmpl.Execute(writer, data)

	}

	processAnswer := func() {

		request.ParseForm()

		userAnswer := request.FormValue("answer")
		cardID, _ := strconv.ParseInt(request.FormValue("card-id"), 10, 64)

		card, _ := g.getCardByID(uint(cardID))

		if IsAnswerCorrectInLowerCase(userAnswer, card.Answer) {
			g.updateLearningCardByID(uint(card.ID), true)

		} else {
			g.updateLearningCardByID(uint(card.ID), false)
		}

		cards, _ := g.getLearningCardsByDeckID(deck.ID)
		mostDueCard, _ := getMostDueCard(cards)

		var cardAvailable bool

		randomCards, _ := g.getRandomCardsByDeckID(deck.ID, mostDueCard.ID)
		randomCards = append(randomCards, mostDueCard)
		rand.Shuffle(len(randomCards), func(i, j int) {
			randomCards[i], randomCards[j] = randomCards[j], randomCards[i]
		})

		if len(randomCards) > 3 && mostDueCard.ID != 0 {
			cardAvailable = true
		} else {
			cardAvailable = false
		}
		tmpl, _ := template.ParseFiles("./templates/htmx/learning-multiple-choice.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			RandomCards   []Card
			MostDueCard   Card
			CardAvailable bool
		}{
			Title:         "Deck: " + deck.Name,
			Deck:          deck,
			RandomCards:   randomCards,
			MostDueCard:   mostDueCard,
			CardAvailable: cardAvailable,
		}

		tmpl.Execute(writer, data)

	}

	switch request.Method {
	case "GET":
		displayLearning()
	case "POST":
		processAnswer()
	}
}

func (g *GormDB) ReviewMultipleChoiceHandler(writer http.ResponseWriter, request *http.Request) {

	IDString := strings.TrimPrefix(request.URL.Path, "/review-multiple-choice/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))

	displayReview := func() {
		cards, _ := g.getDueReviewCardsByDeckID(deck.ID)
		mostDueCard, _ := getMostDueCard(cards)
		var cardAvailable bool

		randomCards, _ := g.getRandomCardsByDeckID(deck.ID, mostDueCard.ID)
		randomCards = append(randomCards, mostDueCard)
		rand.Shuffle(len(randomCards), func(i, j int) {
			randomCards[i], randomCards[j] = randomCards[j], randomCards[i]
		})

		if len(randomCards) > 3 {
			cardAvailable = true
		} else {
			cardAvailable = false
		}
		tmpl, _ := template.ParseFiles("./templates/htmx/review-multiple-choice.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			RandomCards   []Card
			MostDueCard   Card
			CardAvailable bool
		}{
			Title:         "Deck: " + deck.Name,
			Deck:          deck,
			RandomCards:   randomCards,
			MostDueCard:   mostDueCard,
			CardAvailable: cardAvailable,
		}

		tmpl.Execute(writer, data)

	}

	processAnswer := func() {

		request.ParseForm()

		userAnswer := request.FormValue("answer")
		cardID, _ := strconv.ParseInt(request.FormValue("card-id"), 10, 64)

		card, _ := g.getCardByID(uint(cardID))

		if IsAnswerCorrectInLowerCase(userAnswer, card.Answer) {
			g.updateReviewCardByID(uint(card.ID), true)

		} else {
			g.updateReviewCardByID(uint(card.ID), false)
		}

		cards, _ := g.getDueReviewCardsByDeckID(deck.ID)
		mostDueCard, _ := getMostDueCard(cards)

		var cardAvailable bool

		randomCards, _ := g.getRandomCardsByDeckID(deck.ID, mostDueCard.ID)
		randomCards = append(randomCards, mostDueCard)
		rand.Shuffle(len(randomCards), func(i, j int) {
			randomCards[i], randomCards[j] = randomCards[j], randomCards[i]
		})

		if len(randomCards) > 3 && mostDueCard.ID != 0 {
			cardAvailable = true
		} else {
			cardAvailable = false
		}
		tmpl, _ := template.ParseFiles("./templates/htmx/review-multiple-choice.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			RandomCards   []Card
			MostDueCard   Card
			CardAvailable bool
		}{
			Title:         "Deck: " + deck.Name,
			Deck:          deck,
			RandomCards:   randomCards,
			MostDueCard:   mostDueCard,
			CardAvailable: cardAvailable,
		}

		tmpl.Execute(writer, data)

	}

	switch request.Method {
	case "GET":
		displayReview()
	case "POST":
		processAnswer()
	}
}

func (g *GormDB) LearningTypingHandler(writer http.ResponseWriter, request *http.Request) {
	//create string without /learning/ from the URL path
	IDString := strings.TrimPrefix(request.URL.Path, "/learning-typing/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))
	cards, _ := g.getLearningCardsByDeckID(deck.ID)
	mostDueCard, _ := getMostDueCard(cards)

	var cardAvailable bool

	if len(cards) > 0 {
		cardAvailable = true
	} else {
		cardAvailable = false
	}

	//GET
	displayCards := func() {
		tmpl, _ := template.ParseFiles("./templates/htmx/learning-typing.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			Cards         []Card
			Card          Card
			CardAvailable bool
		}{
			Title:         "Learning session for " + deck.Name,
			Deck:          deck,
			Cards:         cards,
			Card:          mostDueCard,
			CardAvailable: cardAvailable,
		}
		tmpl.Execute(writer, data)
	}

	//POST
	processAnswer := func() {
		request.ParseForm()

		userAnswer := request.FormValue("answer")
		cardID, _ := strconv.ParseInt(request.FormValue("card-id"), 10, 64)

		card, _ := g.getCardByID(uint(cardID))

		if IsAnswerCorrectInLowerCase(userAnswer, card.Answer) {
			g.updateLearningCardByID(uint(card.ID), true)
			cards, _ := g.getLearningCardsByDeckID(deck.ID)
			mostDueCard, _ := getMostDueCard(cards)

			if len(cards) > 0 {

				data := struct {
					Title         string
					Deck          Deck
					Cards         []Card
					Card          Card
					CardAvailable bool
				}{
					Title:         "Learning session for " + deck.Name,
					Deck:          deck,
					Cards:         cards,
					Card:          mostDueCard,
					CardAvailable: cardAvailable,
				}

				tmpl, _ := template.ParseFiles("./templates/htmx/learning-typing.html")

				tmpl.Execute(writer, data)

			} else {
				data := struct {
					Message string
				}{
					Message: "No learning cards left for this deck. Create some new cards ",
				}
				tmpl, _ := template.ParseFiles("./templates/htmx/nocards.html")

				tmpl.Execute(writer, data)
			}

		} else {
			g.updateLearningCardByID(uint(card.ID), false)
			data := struct {
				Question      string
				UserAnswer    string
				CorrectAnswer string
				Route         string
			}{
				Question:      card.Question,
				UserAnswer:    userAnswer,
				CorrectAnswer: card.Answer,
				Route:         "/learning-typing/" + IDString,
			}
			tmpl, _ := template.ParseFiles("./templates/htmx/wrong-answer.html")

			tmpl.Execute(writer, data)
		}

	}

	switch request.Method {
	case "GET":
		displayCards()
	case "POST":
		processAnswer()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}
func (g *GormDB) ReviewTypingHandler(writer http.ResponseWriter, request *http.Request) {
	//create string without /learning/ from the URL path
	IDString := strings.TrimPrefix(request.URL.Path, "/review-typing/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))
	cards, _ := g.getDueReviewCardsByDeckID(deck.ID)
	mostDueCard, _ := getMostDueCard(cards)

	var cardAvailable bool

	if len(cards) > 0 {
		cardAvailable = true
	} else {
		cardAvailable = false
	}

	//GET
	displayCards := func() {
		tmpl, _ := template.ParseFiles("./templates/htmx/review-typing.html")
		data := struct {
			Title         string
			Deck          Deck
			Cards         []Card
			Card          Card
			CardAvailable bool
		}{
			Title:         "Review session for " + deck.Name,
			Deck:          deck,
			Cards:         cards,
			Card:          mostDueCard,
			CardAvailable: cardAvailable,
		}
		tmpl.Execute(writer, data)
	}

	//POST
	processAnswer := func() {
		request.ParseForm()

		userAnswer := request.FormValue("answer")
		cardID, _ := strconv.ParseInt(request.FormValue("card-id"), 10, 64)

		card, _ := g.getCardByID(uint(cardID))

		if IsAnswerCorrectInLowerCase(userAnswer, card.Answer) {
			g.updateReviewCardByID(uint(card.ID), true)

		} else {
			g.updateReviewCardByID(uint(card.ID), false)
		}

		cards, _ := g.getDueReviewCardsByDeckID(deck.ID)
		mostDueCard, _ := getMostDueCard(cards)

		if len(cards) > 0 {

			data := struct {
				Title         string
				Deck          Deck
				Cards         []Card
				Card          Card
				CardAvailable bool
			}{
				Title:         "Review session for " + deck.Name,
				Deck:          deck,
				Cards:         cards,
				Card:          mostDueCard,
				CardAvailable: cardAvailable,
			}

			tmpl, _ := template.ParseFiles("./templates/htmx/review-typing.html")

			tmpl.Execute(writer, data)

		} else {
			data := struct {
				Message string
			}{
				Message: "No learning cards left for this deck. Create some new cards ",
			}
			tmpl, _ := template.ParseFiles("./templates/htmx/nocards.html")

			tmpl.Execute(writer, data)
		}

	}

	switch request.Method {
	case "GET":
		displayCards()
	case "POST":
		processAnswer()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}

func (g *GormDB) LearningHandler(writer http.ResponseWriter, request *http.Request) {
	IDString := strings.TrimPrefix(request.URL.Path, "/learning/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))
	cards, _ := g.getLearningCardsByDeckID(deck.ID)
	var cardAvailable bool

	if len(cards) > 0 {
		cardAvailable = true
	} else {
		cardAvailable = false
	}

	displayDeckLearning := func() {
		tmpl, _ := template.ParseFiles("./templates/learn.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			CardAvailable bool
		}{
			Title:         "Learning session for " + deck.Name,
			Deck:          deck,
			CardAvailable: cardAvailable,
		}
		tmpl.Execute(writer, data)
	}

	switch request.Method {
	case "GET":
		displayDeckLearning()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}
func (g *GormDB) ReviewHandler(writer http.ResponseWriter, request *http.Request) {
	IDString := strings.TrimPrefix(request.URL.Path, "/review/")
	id, _ := strconv.Atoi(IDString)
	deck, _ := g.getDeckByID(uint(id))
	cards, _ := g.getDueReviewCardsByDeckID(deck.ID)
	var cardAvailable bool

	if len(cards) > 0 {
		cardAvailable = true
	} else {
		cardAvailable = false
	}

	displayDeckLearning := func() {
		tmpl, _ := template.ParseFiles("./templates/review.html", "./templates/navbar.html")
		data := struct {
			Title         string
			Deck          Deck
			CardAvailable bool
		}{
			Title:         "Review session for " + deck.Name,
			Deck:          deck,
			CardAvailable: cardAvailable,
		}
		tmpl.Execute(writer, data)
	}

	switch request.Method {
	case "GET":
		displayDeckLearning()
	default:
		http.Error(writer, "Unsupported method", http.StatusMethodNotAllowed)
	}

}

func (g *GormDB) CreateCardHandler(writer http.ResponseWriter, request *http.Request) {
	displayForm := func() {
		tmpl, _ := template.ParseFiles("./templates/create_card.html", "./templates/navbar.html")
		decks, _ := g.selectAllDecks()
		data := struct {
			Title   string
			Heading string
			Message string
			Decks   []Deck
		}{
			Title:   "Card creation",
			Heading: "Create a card",
			Message: "A card needs a question and an answer",
			Decks:   decks,
		}
		tmpl.Execute(writer, data)
	}
	processForm := func() {
		request.ParseForm()

		deckID, _ := strconv.ParseInt(request.FormValue("deck-id"), 10, 64)
		question := request.FormValue("question")
		answer := request.FormValue("answer")

		t, _ := time.Now().UTC().MarshalText()

		var card Card
		card.DeckID = uint(deckID)
		card.Question = question
		card.Answer = answer
		card.CardCreated = string(t)
		card.ReviewDueDate = string(t) //necessary to avoid a critical error when determing which card to show first for cards that have never been answered before.
		g.createCard(card)

		fmt.Fprintf(writer, "<div id='result'>Card with question '%s' and answer '%s' created successfully!</div>", question, answer)

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

func (g *GormDB) DecksHandler(writer http.ResponseWriter, request *http.Request) {
	displayDecks := func() {
		tmpl, _ := template.ParseFiles("./templates/decks.html", "./templates/navbar.html")
		decks, _ := g.selectAllDecks()
		data := struct {
			Title string
			Decks []Deck
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

func getNextEaseLevel(currentEase int, growthfactor float64) int {
	growthFactor := growthfactor
	nextEase := int(math.Ceil(float64(currentEase) * growthFactor))

	return nextEase
}

func createNextReviewDueDate(ease int) string {

	t := time.Now().UTC()
	t = t.Add(time.Duration(ease) * 24 * time.Hour)

	formattedTime, _ := t.MarshalText()
	return string(formattedTime)
}

func main() {
	fmt.Println(startMessage())

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	gormDB := &GormDB{db: db}

	db.AutoMigrate(&Deck{}, &Card{})

	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/create-deck", gormDB.CreateDeckHandler)
	http.HandleFunc("/decks", gormDB.DecksHandler)
	http.HandleFunc("/learning/", gormDB.LearningHandler)
	http.HandleFunc("/review/", gormDB.ReviewHandler)
	http.HandleFunc("/deck/", gormDB.DeckHandler)
	http.HandleFunc("/create-card", gormDB.CreateCardHandler)
	http.HandleFunc("/learning-typing/", gormDB.LearningTypingHandler)
	http.HandleFunc("/learning-multiple-choice/", gormDB.LearningMultipleChoiceHandler)
	http.HandleFunc("/review-multiple-choice/", gormDB.ReviewMultipleChoiceHandler)
	http.HandleFunc("/review-typing/", gormDB.ReviewTypingHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	fmt.Println("Server starting at :8080")
	http.ListenAndServe(":8080", nil)

}
