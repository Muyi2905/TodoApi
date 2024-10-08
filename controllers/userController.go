package controllers

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/muyi2905/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var validate = validator.New()
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type Claims struct {
	UserId uint `json:"user_id"`
	jwt.StandardClaims
}

func GetUsers(c *gin.Context, db *gorm.DB) {
	var users []models.User

	if err := db.Find(&users).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * pageSize

	// Filtering
	name := c.Query("name")
	email := c.Query("email")

	query := db.Model(&models.User{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if email != "" {
		query = query.Where("email LIKE ?", "%"+email+"%")
	}

	var total int64
	query.Count(&total)

	// Execute query with pagination
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  (int(total) + pageSize - 1) / pageSize,
			"total_items":  total,
		},
	})
}

func CreateUser(c *gin.Context, db *gorm.DB) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}
	if err := validate.Struct(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}

	var existingUser models.User
	err := db.Where("email = ?", user.Email).First(&existingUser).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user alredy exist "})
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created sucessful",
	})
}

func GetUserById(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var user models.User

	if err := db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "user not found"})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
	}
	c.JSON(http.StatusOK, gin.H{
		"user": "user",
	})
}

func UpdateUser(c *gin.Context, db *gorm.DB) {

	id := c.Param("id")

	var updatedUser models.User
	err := c.ShouldBindJSON(&updatedUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"Error": err.Error()})
		return
	}

	if err := validate.Struct(updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := db.Find(&updatedUser, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"err": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		}
	}

	if err := db.Model(&user).Updates(&updatedUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

func DeleteUser(c *gin.Context, db *gorm.DB) {
	id := c.Param("id")

	var user models.User

	if err := db.First(user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"err": "user not found"})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
	}

	if err := db.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

func Signup(c *gin.Context, db *gorm.DB) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	if err := validate.Struct(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return

	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": "failed to hash password"})
		return
	}
	user.Password = string(hashedPassword)

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	claims := &Claims{
		UserId: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})

}

func Login(c *gin.Context, db *gorm.DB) {
	var credentials struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	result := db.Where("email = ?", credentials.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := generateJWTToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func generateJWTToken(userID uint) (string, error) {
	claims := &Claims{
		UserId: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
