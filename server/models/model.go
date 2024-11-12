package models

import "time"

// syn_category

type Category struct {
	Name         string    `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type UpdateCategory struct {
	CategoryID   int    `db:"category_id" json:"category_id"`
	NewName      string    `db:"new_name" json:"new_name"`
	NewDescription  string    `db:"new_description" json:"new_description"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// syn_author

type Author struct {
	Name         string    `db:"name" json:"name"`
	Birthdate    *time.Time `db:"birthdate" json:"birthdate,omitempty"`
	Nationality  *string   `db:"nationality" json:"nationality,omitempty"`
	Biography    *string   `db:"biography" json:"biography,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type UpdateAuthor struct {
	AuthorID     int    `db:"author_id" json:"author_id"`
	NewName      string    `db:"new_name" json:"new_name"`
	NewBirthdate string    `db:"new_birthdate" json:"new_birthdate"`
	NewNationality string    `db:"new_nationality" json:"new_nationality"`
	NewBiography string    `db:"new_biography" json:"new_biography"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// syn_book

type Book struct {
	Title          string    `db:"title" json:"title"`
	CategoryID     int       `db:"category_id" json:"category_id"`
	AuthorID       int       `db:"author_id" json:"author_id"`
	PublishedDate  *time.Time `db:"published_date" json:"published_date,omitempty"`
	ISBN           *string   `db:"isbn" json:"isbn,omitempty"`
	TotalStock     int       `db:"total_stock" json:"total_stock"`
	AvailableStock int       `db:"available_stock" json:"available_stock"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type UpdateBook struct {
	BookID        int       `db:"book_id" json:"book_id"`
	NewTitle      string    `db:"new_title" json:"new_title"`
	NewCategoryID int       `db:"new_category_id" json:"new_category_id"`
	NewAuthorID   int       `db:"new_author_id" json:"new_author_id"`
	NewPublishedDate string    `db:"new_published_date" json:"new_published_date"`
	NewISBN       string    `db:"new_isbn" json:"new_isbn"`
	NewTotalStock int       `db:"new_total_stock" json:"new_total_stock"`
	NewAvailableStock int       `db:"new_available_stock" json:"new_available_stock"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Borrow struct {
	BookID        int      `db:"book_id" json:"book_id"`
	UserID        int       `db:"user_id" json:"user_id"`
	BorrowedDate  time.Time `db:"borrowed_date" json:"borrowed_date"`
	ReturnDate  time.Time `db:"returned_date" json:"returned_date"`
	ReturnedDate    *time.Time `db:"return_date" json:"return_date,omitempty"`
	Returned      bool      `db:"returned" json:"returned"`
}

type UpdateBorrow struct {
	BorrowingID   int       `db:"borrowing_id" json:"borrowing_id"`
	NewBookID     int       `db:"new_book_id" json:"new_book_id"`
	NewUserID     int       `db:"new_user_id" json:"new_user_id"`
	NewBorrowedDate string    `db:"new_borrowed_date" json:"new_borrowed_date"`
	NewReturnDate string    `db:"new_return_date" json:"new_return_date"`
	NewReturnedDate string    `db:"new_returned_date" json:"new_returned_date"`
	NewReturned   string      `db:"new_returned" json:"new_returned"`
}

// syn_user

type UserSensitive struct {
	Username     string    `db:"username" json:"username"`
	Password	 string    `db:"password_hash" json:"-"`
	FirstName    *string   `db:"first_name" json:"first_name,omitempty"`
	LastName     *string   `db:"last_name" json:"last_name,omitempty"`
	Email        string    `db:"email" json:"email"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type User struct {
	UserID       int       `db:"user_id" json:"user_id"`
	Username     string    `db:"username" json:"username"`
	FirstName    *string   `db:"first_name" json:"first_name,omitempty"`
	LastName     *string   `db:"last_name" json:"last_name,omitempty"`
	Email        string    `db:"email" json:"email"`
	Role         string    `db:"role" json:"role"`
}

type UserPassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserIDPassword struct {
	UserID   int    `json:"user_id"`
	Password string `json:"password"`
}

type NewPassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
	NewPassword string `json:"new_password"`
}