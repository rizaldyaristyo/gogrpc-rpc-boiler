#!/bin/bash

# Database connection variables
DB_HOST="db"
DB_USER="postgres"

# Function to check if a database exists
database_exists() {
    psql -h "$DB_HOST" -U "$DB_USER" -tc "SELECT 1 FROM pg_database WHERE datname = '$1'" | grep -q 1
}

# Function to create a database if it does not exist
create_database_if_not_exists() {
    local db_name="$1"
    if ! database_exists "$db_name"; then
        echo "Creating database: $db_name"
        psql -h "$DB_HOST" -U "$DB_USER" -c "CREATE DATABASE $db_name"
    else
        echo "Database $db_name already exists"
    fi
}

# Function to check if a table exists in a specific database
table_exists() {
    local db_name="$1"
    local table_name="$2"
    psql -h "$DB_HOST" -U "$DB_USER" -d "$db_name" -tc "SELECT 1 FROM information_schema.tables WHERE table_name = '$table_name'" | grep -q 1
}

# Function to create a table if it does not exist
create_table_if_not_exists() {
    local db_name="$1"
    local table_name="$2"
    local create_query="$3"
    if ! table_exists "$db_name" "$table_name"; then
        echo "Creating table: $table_name in database: $db_name"
        psql -h "$DB_HOST" -U "$DB_USER" -d "$db_name" -c "$create_query"
    else
        echo "Table $table_name already exists in database $db_name"
    fi
}

# Create databases if they do not exist
create_database_if_not_exists "syn_category"
create_database_if_not_exists "syn_author"
create_database_if_not_exists "syn_user"
create_database_if_not_exists "syn_book"

# Table creation queries
AUTHOR_TABLE_QUERY="CREATE TABLE authors (
    author_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    birthdate DATE,
    nationality VARCHAR(100),
    biography TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);"

CATEGORY_TABLE_QUERY="CREATE TABLE categories (
    category_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);"

USER_TABLE_QUERY="CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(100) NOT NULL UNIQUE,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);"

BOOK_TABLE_QUERY="CREATE TABLE books (
    book_id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    category_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    published_date DATE,
    isbn VARCHAR(13) UNIQUE,
    total_stock INTEGER DEFAULT 0,
    available_stock INTEGER DEFAULT 0 CHECK (available_stock >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);"

BORROWING_TABLE_QUERY="CREATE TABLE borrowing (
    borrowing_id SERIAL PRIMARY KEY,
    book_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    borrowed_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    return_date TIMESTAMP,
    returned BOOLEAN DEFAULT FALSE,
    returned_date TIMESTAMP
);"

# Create tables if they do not exist
create_table_if_not_exists "syn_author" "authors" "$AUTHOR_TABLE_QUERY"
create_table_if_not_exists "syn_category" "categories" "$CATEGORY_TABLE_QUERY"
create_table_if_not_exists "syn_user" "users" "$USER_TABLE_QUERY"
create_table_if_not_exists "syn_book" "books" "$BOOK_TABLE_QUERY"
create_table_if_not_exists "syn_book" "borrowing" "$BORROWING_TABLE_QUERY"
