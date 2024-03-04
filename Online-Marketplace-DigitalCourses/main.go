package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-database/mysql"
)

var (
	db *sql.DB
)

func init() {
	var err error
	db, err := sql.Open("mysql", "joshua468:170821002@tcp(localhost:3306)/mydb")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`
CREATE TABLE IF NOT Exixts users (
	username VARCHAR(50);PRIMARY KEY,
	password VARCHAR(50),
	role VARCHAR(50),
	)
   `)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(` 
   CREATE TABLE IF NOT Exists courses (
	id INT AUTO_INCREMENT PRIMARY KEY,
	title 		VARCHAR(100),
	description VARCHAR(255),
	Price 		DECIMAL(10,2),
	duration 	VARCHAR(50),
	instructor	VARCHAR(50),
	created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   )
  `)
	_, err = db.Exec(`
	CREATE TABLE IF NOT Exists enrollments (
		username VARCHAR(50),
		course_id INT,
		enrolled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
		PRIMARY_KEY(username,enrolled_id),
		FOREIGN KEY(username)REFERENCES users(username),
		FOREIGN KEY(course_id) REFERNCES courses(id)

	)
	`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT Exists payments(
		id INT AUTO_INCREMENT PRIMARY KEY,
		usernname VARCHAR(50),
		course_id INT,
		amount DECIMAL(10,2),
		paid_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (username) REFERENCES users(username),
		FOREIGN KEY (username) ReFERENCES courses(course_id)
	)
	`)

	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT Exists progress(
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50),
		course_id INT,
		completed_lectures INT,
		quiz_score DECIMAL(5,2),
		completed BOOLEAN,
		completed_at TIMESTAMP,
		FOREIGN KEY (username)REFRENCES users(username),
		FOREIGN KEY (course_id) REFRENCES courses(id)
	)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Course struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Duration    string    `json:"duration"`
	Instructor  string    `json:"instructor"`
	CreatedAt   time.Time `json:"created_at"`
}

type Enrollment struct {
	Username   int       `json:"username"`
	CourseID   int       `json:"course_id"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

type Payment struct {
	ID       int     `json:"id"`
	Username string  `json:"username"`
	CourseID int     `json:"courseid"`
	Amount   float64 `json:"amount"`
}

type Progress struct {
	ID                int       `json:"id"`
	Username          string    `json:"username"`
	CourseID          int       `json:"course_id"`
	CompletedLectures int       `json:"completed_lectures"`
	Quiz_score        float64   `json:"quiz_score"`
	Completed         bool      `json:"completed"`
	CompletedAt       time.Time `json:"completed_at"`
}

func main() {
	r := gin.Default()
	r.POST("/login", Loginhandler)
	instructorGroup := r.Group("/instructor", authMiddleware("/instructor"))
	{
		instructorGroup.POST("/courses", createCourseHandler)
		instructorGroup.PUT("/courses:id", updateCourseHandler)
		instructorGroup.DELETE("/courses:id", deleteCourseHandler)

	}
	studentGroup := r.Group("/student", authMiddleware("/student"))
	{
		studentGroup.POST("/enrollments:id", enrollCourseHandler)
		studentGroup.GET("/enrollments", getEnrollmentsHandler)
		studentGroup.POST("/payments/:id", makePaymentHandler)
		studentGroup.POST("/progress", updateProgressHandler)
	}
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func loginHandler(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var storeduser User
	err := db.QueryRow("SELECT username, role FROM users WHERE username =? AND password =?", storeduser.Username, storeduser.Password).Scan(&storeduser.Username, &storeduser.Role)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodH256, jwtMapClaims{
		"username": &storeduser.Username,
		"role":     &storeduser.Role,
	})
	tokenString, err := token.SignedString(secretkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func authMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		claims, ok := token.Claims(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}
		roleClaim, ok := claims["role"].(string)
		if !ok || roleClaim != role {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}
		c.Next()
	}
}
func createCourseHandler(c *gin.Context) {
	var course Course
	if err := c.BindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := db.Exec("INSERT INTO courses(titles,description,price,duration,instructor) VALUES(?,?,?,?,?)", course.Title, course.Description, course.Price, course.Duration, course.Instructor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	course.ID = int(id)
	course.CreatedAt = time.Now()
	c.JSON(http.StatusCreated, course)
}

func updateCourseHandler(c *gin.Context) {
	id := c.Params("id")
	var course Course
	if err := c.BindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec("UPDATE courses SET title=?,description=?,price=?,duration=?,instructor=? WHERE id=?", course.Title, course.Description, course.Price, course.Duration, course.Instructor, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

}

func deleteCourseHandler(c *gin.Context) {
	id := c.Params("id")
	_, err := db.Exec("DELETE FROM courses WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Course deleted successfully"})
}

func enrollCourseHandler(c *gin.Context) {
	id := c.Params("id")
	username := getUsernameFromToken(c)
	_, err := db.Exec("INSERT INTO enrollments (username,course_id) VALUES(??)", username, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	enrollment := Enrollment{Username: username, CourseID: id, EnrolledAt: time.Now()}
	c.JSON(http.StatusCreated, enrollment)
}

func getEnrollmentsHandler(c *gin.Context) {
	username := getUsernameFromToken(c)
	rows, err := db.Query("SELECT username,course_id,enrolled_at FROM enrollments WHERE username=?", username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var enrollments []Enrollment
	for rows.Next() {
		var enrollment Enrollment
		if err := rows.Scan(&enrollment.Username, &enrollment.CourseID, &enrollment.EnrolledAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		enrollments = append(enrollments, enrollment)
	}

	c.JSON(http.StatusOK, enrollments)
}

func makePaymentHandler(c *gin.Context) {
	id := c.Param("id")
	username := getUsernameFromToken(c)

	var course Course
	err := db.QueryRow("SELECT id, price FROM courses WHERE id=?", id).Scan(&course.ID, &course.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec("INSERT INTO payments (username, course_id, amount) VALUES (?, ?, ?)", username, id, course.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	payment := Payment{Username: username, CourseID: course.ID, Amount: course.Price, PaidAt: time.Now()}
	c.JSON(http.StatusCreated, payment)
}

func updateProgressHandler(c *gin.Context) {
	id := c.Param("id")
	username := getUsernameFromToken(c)

	var progress Progress
	if err := c.BindJSON(&progress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("INSERT INTO progress (username, course_id, completed_lectures, quiz_score, completed, completed_at) VALUES (?, ?, ?, ?, ?, ?)", username, id, progress.CompletedLectures, progress.QuizScore, progress.Completed, progress.CompletedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	progress.ID = int(id)
	c.JSON(http.StatusCreated, progress)
}

func getUsernameFromToken(c *gin.Context) string {
	tokenString := c.GetHeader("Authorization")
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	claims, _ := token.Claims.(jwt.MapClaims)
	username, _ := claims["username"].(string)
	return username
}
