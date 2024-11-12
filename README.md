# API Documentation

### Base URL

## All endpoints are based on: `http://localhost:3000`

# Endpoints

## **Create User**

-   ### **POST** `/createuser`
    -   **Description**: Registers a new user with provided details.
    -   **Parameters** (form data):
        -   `username` (string)
        -   `password` (string)
        -   `first_name` (string)
        -   `last_name` (string)
        -   `email` (string)
        -   `role` (string)

## **Login**

-   ### **POST** `/login`
    -   **Description**: Authenticates a user.
    -   **Parameters** (form data):
        -   `username` (string)
        -   `password` (string)

## **Change Password**

-   ### **POST** `/changepassword`
    -   **Description**: Changes the password for a user.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `username` (string)
        -   `password` (string)
        -   `new_password` (string)

## **Get User**

-   ### **GET** `/getuser/{id}`
    -   **Description**: Retrieves details for a specific user.
    -   **Authorization**: Bearer token required.

## **Delete User**

-   ### **POST** `/deleteuser`
    -   **Description**: Deletes a user account.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `user_id` (string)
        -   `password` (string)

---

# Author Endpoints

**Create Author**

-   ### **POST** `/createauthor`
    -   **Description**: Registers a new author.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `name` (string)
        -   `birthdate` (date)
        -   `nationality` (string)
        -   `biography` (string)

## **Get Authors**

-   ### **POST** `/getauthors`
    -   **Description**: Retrieves a list of authors.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `min` (int)
        -   `max` (int)

## **Search Authors by Name**

-   ### **POST** `/getauthorsbyname`
    -   **Description**: Retrieves authors matching a specific name.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `name` (string)

## **Get Author by ID**

-   ### **GET** `/getauthorbyid/{id}`
    -   **Description**: Retrieves an author's details by ID.
    -   **Authorization**: Bearer token required.

## **Edit Author**

-   ### **POST** `/editauthor`
    -   **Description**: Updates author information.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `author_id` (string)
        -   `new_name`, `new_birthdate`, `new_nationality`, `new_biography` (string)

## **Delete Author**

-   ### **POST** `/deleteauthor`
    -   **Description**: Deletes an author.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `author_id` (string)

---

# Category Endpoints

**Create Category**

-   ### **GET** `/createcategory`
    -   **Description**: Registers a new category.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `name` (string)
        -   `description` (string)

## **Get Categories**

-   ### **GET** `/getcategories`
    -   **Description**: Retrieves a list of categories.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `min`, `max` (int)

## **Search Categories by Name**

-   ### **POST** `/getcategoriesbyname`
    -   **Description**: Searches categories by name.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `name` (string)

## **Get Category by ID**

-   ### **GET** `/getcategorybyid/{id}`
    -   **Description**: Retrieves a category by ID.
    -   **Authorization**: Bearer token required.

## **Edit Category**

-   ### **POST** `/editcategory`
    -   **Description**: Updates category information.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `category_id` (string)
        -   `new_name`, `new_description` (string)

## **Delete Category**

-   ### **POST** `/deletecategory`
    -   **Description**: Deletes a category.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `category_id` (string)

---

# Book Endpoints

**Create Book**

-   ### **POST** `/createbook`
    -   **Description**: Registers a new book.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `title`, `category_id`, `author_id`, `published_date`, `isbn`, `total_stock`, `available_stock`

## **Get Books**

-   ### **POST** `/getbooks`
    -   **Description**: Retrieves a list of books.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `min`, `max` (int)

## **Get Book by ID**

-   ### **GET** `/getbookbyid/{id}`
    -   **Description**: Retrieves a book by ID.
    -   **Authorization**: Bearer token required.

## **Edit Book**

-   ### **POST** `/editbook`
    -   **Description**: Updates book information.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `book_id`, `new_title`, `new_category_id`, `new_author_id`, `new_published_date`, `new_isbn`, `new_total_stock`, `new_available_stock`

## **Delete Book**

-   ### **POST** `/deletebook`
    -   **Description**: Deletes a book.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `book_id` (string)

---

# Additional Endpoints

---

### **User Existence**

**Check User Existence**

-   ### **GET** `/doesuserexist/{id}`
    -   **Description**: Checks if a user exists.
    -   **Authorization**: Bearer token required.

### Author Existence

**Check Author Existence**

-   ### **GET** `/doesauthorexist/{id}`
    -   **Description**: Checks if an author exists.
    -   **Authorization**: Bearer token required.
    

### Category Existence

**Check Category Existence**

-   ### **GET** `/doescategoryexist`
    -   **Description**: Checks if a category exists.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `category_id` (string)

### Usage in Books

**Check if Author is Used in Books**

-   ### **GET** `/isauthorinusebybook/{id}`
    -   **Description**: Verifies if an author is associated with any books.
    -   **Authorization**: Bearer token required.

## **Check if Category is Used in Books**

-   ### **GET** `/iscategoryinusebybook/{id}`
    -   **Description**: Verifies if a category is associated with any books.
    -   **Authorization**: Bearer token required.

---

### Book Queries

**Get Books by Date**

-   ### **POST** `/getbooksbydate`
    -   **Description**: Retrieves books within a date range.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `start_date` (date)
        -   `end_date` (date)

## **Search Books by Name**

-   ### **POST** `/getbooksbyname`
    -   **Description**: Searches for books matching a name.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `name` (string)

---

### Borrowing Management

**Check if User Still Borrows**

-   ### **POST** `/doesuserstillborrow/{id}`
    -   **Description**: Checks if a user is still borrowing a book.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `user_id` (string)

## **Create Borrow Record**

-   ### **POST** `/createborrow`
    -   **Description**: Registers a new borrowing transaction.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `book_id`, `user_id`, `borrowed_date`, `return_date`, `returned` (boolean)

## **Create Return Record**

-   ### **POST** `/createreturn`
    -   **Description**: Registers a return for a borrowed book.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `borrow_id` (string)

## **Get Borrowings**

-   ### **POST** `/getborrowings`
    -   **Description**: Retrieves all borrowing records.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `min`, `max` (int)

## **Get Borrowings by Date**

-   ### **POST** `/getborrowingsbydate`
    -   **Description**: Retrieves borrowing records within a date range.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `start_date` (date)
        -   `end_date` (date)

## **Get Borrowings by User ID**

-   ### **GET** `/getborrowingsbyuserid/{id}`
    -   **Description**: Retrieves all borrowings by a specific user.
    -   **Authorization**: Bearer token required.

---

### Return Management

**Get Returns**

-   ### **POST** `/getreturns`
    -   **Description**: Retrieves all return records.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `min`, `max` (int)

## **Get Returns by Date**

-   ### **POST** `/getreturnsbydate`
    -   **Description**: Retrieves return records within a date range.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `start_date` (date)
        -   `end_date` (date)

## **Get Returns by User ID**

-   ### **GET** `/getreturnsbyuserid/{id}`
    -   **Description**: Retrieves return records for a specific user.
    -   **Authorization**: Bearer token required.

---

### Overdue Management

**Get Overdues**

-   ### **POST** `/getoverdues`
    -   **Description**: Retrieves all overdue borrowings within a date range.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `start_date` (date)
        -   `end_date` (date)

---

### Edit and Delete Borrowing

**Edit Borrow Record**

-   ### **POST** `/editborrow`
    -   **Description**: Updates an existing borrow record.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `borrowing_id`, `new_book_id`, `new_user_id`, `new_borrowed_date`, `new_return_date`, `new_returned_date`, `new_returned`

## **Delete Borrow Record**

-   ### **POST** `/deleteborrow`
    -   **Description**: Deletes a borrow record.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `borrowing_id` (string)

---

### Book Recommendations

**Get Book Recommendations**

-   ### **POST** `/getbookrecommendations`
    -   **Description**: Provides book recommendations based on category.
    -   **Authorization**: Bearer token required.
    -   **Parameters** (form data):
        -   `category_id` (string)
        -   `limit` (int)

---
