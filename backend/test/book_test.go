package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"log"

	"github.com/swapxs/LibMS/backend/controllers"
	"github.com/swapxs/LibMS/backend/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

)
// setupAuthTestDB initializes an in-memory SQLite database and seeds test data.
func setupAuthTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// Migrate models
	db.AutoMigrate(&models.User{}, &models.Library{}, &models.BookInventory{}, &models.RequestEvent{}, &models.IssueRegistry{})

	// Create test library
	library := models.Library{Name: "Test Library"}
	db.Create(&library)

	// Hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	// Create test admin user
	adminUser := models.User{
		Name:          "Admin User",
		Email:         "admin@example.com",
		Password:      string(hashedPassword),
		ContactNumber: "1234567890",
		Role:          "LibraryAdmin",
		LibraryID:     library.ID,
	}
	db.Create(&adminUser)

	// Create test reader user
	readerUser := models.User{
		Name:          "Reader User",
		Email:         "reader@example.com",
		Password:      string(hashedPassword),
		ContactNumber: "1234567890",
		Role:          "Reader",
		LibraryID:     library.ID,
	}
	db.Create(&readerUser)

	// Create a test book
	book := models.BookInventory{
		ISBN:            "1234567890",
		LibraryID:       library.ID,
		Title:           "Sample Book",
		Author:          "Author One",
		Publisher:       "Test Publisher",
		Language:        "English",
		Version:         "1st",
		TotalCopies:     5,
		AvailableCopies: 5,
	}
	db.Create(&book)

	return db
}

// ✅ Fix: Use `jwt.MapClaims` instead of `map[string]interface{}`
func addJWTAuthMiddleware(router *gin.Engine, user models.User) {
	router.Use(func(c *gin.Context) {
		claims := jwt.MapClaims{
			"id":         float64(user.ID),
			"email":      user.Email,
			"role":       user.Role,
			"library_id": float64(user.LibraryID),
		}
		c.Set("user", claims)
		c.Next()
	})
}

func TestAddBook(t *testing.T) {
	db := setupAuthTestDB()
	router := gin.Default()

	var adminUser models.User
	db.Where("role = ?", "LibraryAdmin").First(&adminUser)
	addJWTAuthMiddleware(router, adminUser)

	router.POST("/books", controllers.AddOrIncrementBook(db))

	// ✅ Use a completely new ISBN to avoid conflict
	input := controllers.AddBookInput{
		ISBN:      "9999999999", // New ISBN
		Title:     "New Book",
		Author:    "New Author",
		Publisher: "New Publisher",
		Version:   "1st",
		Copies:    3,
	}
	jsonValue, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/books", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	// Capture response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
