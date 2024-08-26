package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func createTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Create tables here for your tests
	createCardsTable(db)
	createDecksTable(db)

	return db, func() {
		db.Close()
	}
}

func TestCreateCardsTable(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	err := createCardsTable(db)
	if err != nil {
		t.Errorf("failed to create cards table: %v", err)
	}

	// Add more assertions to verify table structure if needed
	// For example, you could query the database to check if the table exists
}

func TestSelectAllDecks(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	// Insert some test decks
	_, err := db.Exec("INSERT INTO decks (deck_name) VALUES ('Deck 1'), ('Deck 2')")
	if err != nil {
		t.Fatalf("failed to insert test decks: %v", err)
	}

	decks, err := selectAllDecks(db)
	if err != nil {
		t.Errorf("selectAllDecks failed: %v", err)
	}

	if len(decks) != 2 {
		t.Errorf("expected 2 decks, got %d", len(decks))
	}
}

func TestSelectDeckByID(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	// Insert a test deck
	_, err := db.Exec("INSERT INTO decks (deck_name) VALUES ('Test Deck')")
	if err != nil {
		t.Fatalf("failed to insert test deck: %v", err)
	}

	deck, err := selectDeckByID(db, 1)
	if err != nil {
		t.Errorf("selectDeckByID failed: %v", err)
	}

	if deck.DeckID != 1 || deck.Deckname != "Test Deck" {
		t.Errorf("incorrect deck returned: %v", deck)
	}
}

func TestInsertDeck(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	err := insertDeck(db, "New Deck")
	if err != nil {
		t.Errorf("insertDeck failed: %v", err)
	}

	// Add assertions to verify the deck was inserted correctly
	// For example, you could select the deck and check its details
}

func TestInsertCard(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	err := insertCard(db, 0, "水", "water", "2024/08/08")
	if err != nil {
		t.Errorf("insertCard failed: %v", err)
	}
}

func TestSelectCardsFromDeck(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	//first need to insert deck, then 2 cards, then select
	//also update cards and verify changes

	insertDeckError := insertDeck(db, "test")

	if insertDeckError != nil {
		t.Errorf("insertDeck failed: %v", insertDeckError)
	}

	err := insertCard(db, 0, "水", "water", time.Now().UTC().String())
	if err != nil {
		t.Errorf("insertCard failed: %v", err)
	}
	err2 := insertCard(db, 0, "火", "fire", time.Now().UTC().String())
	if err2 != nil {
		t.Errorf("insertCard failed: %v", err2)
	}

	cards, selectError := selectAllCardsByDeckID(db, 0)

	if selectError != nil {
		t.Errorf("SelectCardsFromDeck failed. Error: %v", selectError)
	}

	if len(cards) == 2 {
		for i := 0; i < len(cards); i++ {
			t.Logf("card question: %v", cards[i].Question)
		}
	} else {
		t.Errorf("Number of expected cards does not match number of cards. Expected: %v, got %v", 2, len(cards))
	}

	updateLearningCard(db, cards[0].CardID, true)
	updateLearningCard(db, cards[1].CardID, false)

	card0, err3 := selectCardByCardID(db, cards[0].CardID)
	if err3 != nil {
		t.Errorf("selectCardByCardID failed: %v", err3)
	}
	card1, err3 := selectCardByCardID(db, cards[1].CardID)
	if err3 != nil {
		t.Errorf("selectCardByCardID failed: %v", err3)
	}

	t.Logf("card0: %#v", card0)
	t.Logf("card0: %#v", card1)

}

func TestInvalidCards(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	cards, err := selectAllCardsByDeckID(db, -1)
	if err != nil {
		t.Errorf("selectAllCardsByDeckID failed with error: %v", err)
	}

	if len(cards) != 0 {
		t.Errorf("Expected a cards length of %v, got %v", 0, len(cards))
	}

}

func TestDeleteDeck(t *testing.T) {
	db, closeDB := createTestDB(t)
	defer closeDB()

	insertDeck(db, "Delete This")
	deck, err := selectDeckByID(db, 1)
	if err != nil {
		t.Fatalf("failed to select test deck: %v", err)
	}

	t.Logf("deck: %v", deck)

	deleteDeck(db, 1)

	deck, err = selectDeckByID(db, 1)
	if err == nil {
		t.Fatalf("Expected an error and no rows, got: %v", deck)
	} else {
		t.Logf("Successfully failed to select deleted deck: %v", err)
	}
}
