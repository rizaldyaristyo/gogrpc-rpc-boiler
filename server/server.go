package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	proto "gogrpc-rpc-boiler/proto"
	database "gogrpc-rpc-boiler/server/db"
	jwtgenerator "gogrpc-rpc-boiler/server/jwt"
	logger "gogrpc-rpc-boiler/server/log"
	"gogrpc-api-boiler/server/models"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	// "github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
    proto.UnimplementedUtilServiceServer
    proto.UnimplementedUserServiceServer
    proto.UnimplementedAuthorServiceServer
    proto.UnimplementedCategoryServiceServer
    proto.UnimplementedBookAndBorrowServiceServer
}

var interServiceConn *grpc.ClientConn
func InitGRPCConnection() {
    logger.LogThis("[INFO] connecting to gRPC server")
    var err error
    interServiceConn, err = grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
    if err != nil {
        logger.LogThis(fmt.Sprintf("[FATAL] failed to connect to gRPC server: %v", err))
    }
    logger.LogThis("[INFO] connected to gRPC server")
}

func CloseGRPCConnection() {
    logger.LogThis("[INFO] closing gRPC connection")
    if interServiceConn != nil {
        interServiceConn.Close()
    } else {
        logger.LogThis("[FATAL] no grpc connection to close")
    }
    logger.LogThis("[INFO] closed gRPC connection")
}

// JWT protector
func validateJWT(ctx context.Context) (string, error) {
    // extract metadata from context
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] missing metadata")
        return "", fmt.Errorf("missing metadata")
    }

    // get token from "authorization" field in metadata
    tokenString := md["authorization"]
    if len(tokenString) == 0 {
        logger.LogThis("[ERROR] authorization token missing")
        return "", fmt.Errorf("authorization token missing")
    }

    // strip token from "Bearer"
    if len(tokenString) > 0 {
        parts := strings.Split(tokenString[0], " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            logger.LogThis("[ERROR] invalid token")
            return "", fmt.Errorf("invalid token")
        }
        tokenString[0] = parts[1]
    }

    // validate JWT
    token, err := jwt.Parse(tokenString[0], func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    if err != nil || !token.Valid {
        logger.LogThis(fmt.Sprintf("[ERROR] invalid token, error: %v, received token: %s", err, tokenString[0]))
        // return fmt.Errorf("invalid token, error: %v, received token: %s", err, tokenString[0])
        logger.LogThis("[ERROR] invalid token")
        return "", fmt.Errorf("invalid token")
    }

    username := token.Claims.(jwt.MapClaims)["username"].(string)

    // debug
    // logger.LogThis(fmt.Sprintf("[INFO] username: %s", username))
    // logger.LogThis(fmt.Sprintf("[INFO] token: %s", tokenString[0]))

    return username, nil
}

// model struct validator
var validate *validator.Validate
func ModelValidator(model interface{}) error {
	validate = validator.New()
	err := validate.Struct(model)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) CreateUser(ctx context.Context, req *proto.UserSensitive) (*proto.StringResponse, error) {

    user := models.UserSensitive{
		Username:  "",
		Password:  "",
		FirstName: new(string),
		LastName:  new(string),
		Email:     "",
		Role:      "",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

    user.Username = req.Username
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to hash password: %v", err))
        return nil, fmt.Errorf("failed to hash password: %v", err)
    }
    user.Password = string(hashedPassword)
    user.FirstName = new(string)
    *user.FirstName = req.FirstName
    user.LastName = new(string)
    *user.LastName = req.LastName
    user.Email = req.Email
    user.Role = req.Role
    user.CreatedAt = time.Now()
    user.UpdatedAt = time.Now()

    // check each field
    if user.Username == "" || user.Password == "" || user.Email == "" || user.Role == "" {
        logger.LogThis("[ERROR] username, password, email, and role are required [Insufficient Input]")
        return nil, fmt.Errorf("username, password, email, and role are required [Insufficient Input]")
    }

    // check if user already exists [Duplicate Entry]
    var exists int
    err = database.UserDB.QueryRow("SELECT 1 AS exists FROM users WHERE username = $1", user.Username).Scan(&exists)
    if err != sql.ErrNoRows && err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check if user exists: %v", err))
        return nil, fmt.Errorf("failed to check if user exists: %v", err)
    }
    if exists == 1 {
        logger.LogThis("[ERROR] user already exists [Duplicate Entry]")
        return nil, fmt.Errorf("user already exists [Duplicate Entry]")
    }


    // insert user into database
    tx, err := database.UserDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [UserDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [UserDB]", err)
    }
    
    _, err = tx.Exec("INSERT INTO users (username, password_hash, first_name, last_name, email, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
        user.Username, user.Password, user.FirstName, user.LastName, user.Email, user.Role, user.CreatedAt, user.UpdatedAt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to insert user: %v", err))
        return nil, fmt.Errorf("failed to insert user: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }
    
    return &proto.StringResponse{ResponseStr: "User created successfully"}, nil
}

func (s *server) LoginAuth(ctx context.Context, req *proto.UserPassword) (*proto.StringResponse, error) {


    user := models.UserPassword{
        Username: "",
        Password: "",
    }

    user.Username = req.Username
    user.Password = req.Password

    row := database.UserDB.QueryRow("SELECT password_hash FROM users WHERE username = $1", user.Username)
    // check if user already exists [Duplicate Entry]
    var storedPassword string
    err := row.Scan(&storedPassword)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    // compare password
    err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password))
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    token, err := jwtgenerator.GenerateJWT(user.Username)
    if err != nil {
        return nil, err
    }
    return &proto.StringResponse{ResponseStr: "Bearer " + token}, nil
}

func (s *server) ChangePassword(ctx context.Context, req *proto.NewPassword) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    user := models.NewPassword{
        Username: "",
        Password: "",
        NewPassword: "",
    }

    user.Username = req.Username
    user.Password = req.Password
    user.NewPassword = req.NewPassword

    // check each field
    if user.Username == "" || user.Password == "" || user.NewPassword == "" {
        logger.LogThis("[ERROR] username, password, and new password are required [Insufficient Input]")
        return nil, fmt.Errorf("username, password, and new password are required [Insufficient Input]")
    }

    row := database.UserDB.QueryRow("SELECT password_hash FROM users WHERE username = $1", user.Username)
    // check if user already exists [Duplicate Entry]
    var storedPassword string
    err := row.Scan(&storedPassword)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    // compare password
    err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password))
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    tx, err := database.UserDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [UserDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [UserDB]", err)
    }

    _, err = tx.Exec("UPDATE users SET password_hash = $1 WHERE username = $2", hashedPassword, user.Username)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to update password: %v", err))
        return nil, fmt.Errorf("failed to update password: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "Password changed successfully"}, nil
}

func (s *server) DeleteUser(ctx context.Context, req *proto.UserIDPassword) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    user := models.UserIDPassword{
    	UserID:   -1,
    	Password: "",
    }

    user.UserID = int(req.UserId)
    user.Password = req.Password

    // check each field
    if user.UserID < 0 || user.Password == "" {
        logger.LogThis("[ERROR] user id and password are required [Insufficient Input]")
        return nil, fmt.Errorf("user id and password are required [Insufficient Input]")
    }

    row := database.UserDB.QueryRow("SELECT password_hash FROM users WHERE user_id = $1", user.UserID)
    // check if user exists
    var storedPassword string
    err := row.Scan(&storedPassword)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    // compare password
    err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password))
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] wrong username or password, error: %v", err))
        return nil, fmt.Errorf("wrong username or password, error: %v", err)
        // return logger.LogThis("[ERROR] wrong username or password")
        // return nil, fmt.Errorf("wrong username or password")
    }

    // check if user still borrows a book, inter-service call to bookservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    bookServiceClient := proto.NewBookAndBorrowServiceClient(interServiceConn)
    doesStillBorrow, err := bookServiceClient.DoesUserStillBorrow(outCtx, &proto.IntRequest{RequestInt: int32(user.UserID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check if user still borrows a book: %v", err))
        return nil, fmt.Errorf("failed to check if user still borrows a book: %v", err)
    }

    if doesStillBorrow.ResponseBool {
        logger.LogThis("[ERROR] user still borrows a book")
        return nil, fmt.Errorf("user still borrows a book")
    }

    // OK Deletes
    tx, err := database.UserDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [UserDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [UserDB]", err)
    }

    _, err = tx.Exec("DELETE FROM users WHERE user_id = $1", user.UserID)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to delete user: %v", err))
        return nil, fmt.Errorf("failed to delete user: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "User deleted successfully"}, nil
}

func (s *server) GetUser(ctx context.Context, req *proto.IntRequest) (*proto.User, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var user models.User
    row := database.UserDB.QueryRow("SELECT user_id, username, first_name, last_name, email, role FROM users WHERE user_id = $1", req.RequestInt,)
    err := row.Scan(&user.UserID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Role)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get user: %v", err))
        return nil, fmt.Errorf("failed to get user: %v", err)
    }

    return &proto.User{
    	UserId:    int32(user.UserID),
    	Username:  user.Username,
    	FirstName: *user.FirstName,
    	LastName:  *user.LastName,
    	Email:     user.Email,
    	Role:      user.Role,
    }, nil
}

func (s *server) DoesUserExist(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var scan int
    err := database.UserDB.QueryRow("SELECT 1 FROM users WHERE user_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        return &proto.BoolResponse{ResponseBool: false}, nil
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check user existence: %v", err))
        return nil, fmt.Errorf("failed to check user existence: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: true}, nil
}

func (s *server) CreateAuthor(ctx context.Context, req *proto.Author) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    parsedBirthDate, err := time.Parse("2006-01-02", req.Birthdate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse birthdate: %v", err))
        return nil, fmt.Errorf("failed to parse birthdate: %v", err)
    }
    author := models.Author{
    	Name:        req.Name,
    	Birthdate:   &parsedBirthDate,
    	Nationality: &req.Nationality,
    	Biography:   &req.Biography,
    	CreatedAt:   time.Now(),
    	UpdatedAt:   time.Now(),
    }

    // validate
    if err := ModelValidator(author); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to validate author: %v", err))
        return nil, fmt.Errorf("failed to validate author: %v", err)
    }

    // check if author is already exists [Duplicate Entry]
    var scan int
    err = database.AuthorDB.QueryRow("SELECT 1 AS exists FROM authors WHERE name = $1 LIMIT 1", author.Name).Scan(&scan)
    if err != sql.ErrNoRows && err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] author already exists [Duplicate Entry], error: %v", err))
        return nil, fmt.Errorf("author already exists [Duplicate Entry], error: %v", err)
    }

    tx, err := database.AuthorDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [AuthorDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [AuthorDB]", err)
    }

    _, err = tx.Exec("INSERT INTO authors (name, birthdate, nationality, biography) VALUES ($1, $2, $3, $4)", author.Name, author.Birthdate, author.Nationality, author.Biography)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to insert author: %v", err))
        return nil, fmt.Errorf("failed to insert author: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "author created successfully"}, nil
}

func (s *server) GetAuthors(ctx context.Context, req *proto.IDLimits) (*proto.AuthorMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var authorMins []*proto.AuthorMin

    rows, err := database.AuthorDB.Query("SELECT author_id, name FROM authors WHERE author_id BETWEEN $1 AND $2", req.Min, req.Max)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get authors: %v", err))
        return nil, fmt.Errorf("failed to get authors: %v", err)
    }

    for rows.Next() {
        var authorMin proto.AuthorMin
        err := rows.Scan(&authorMin.AuthorId, &authorMin.Name)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan author: %v", err))
            return nil, fmt.Errorf("failed to scan author: %v", err)
        }
        authorMins = append(authorMins, &authorMin)
    }

    return &proto.AuthorMins{Authors: authorMins}, nil
}

func (s *server) GetAuthorsByName(ctx context.Context, req *proto.StringRequest) (*proto.AuthorMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var authorMins []*proto.AuthorMin

    // ilike %str%
    rows, err := database.AuthorDB.Query("SELECT author_id, name FROM authors WHERE name ILIKE $1", "%"+req.RequestStr+"%")
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get authors: %v", err))
        return nil, fmt.Errorf("failed to get authors: %v", err)
    }

    for rows.Next() {
        var authorMin proto.AuthorMin
        err := rows.Scan(&authorMin.AuthorId, &authorMin.Name)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan author: %v", err))
            return nil, fmt.Errorf("failed to scan author: %v", err)
        }
        authorMins = append(authorMins, &authorMin)
    }

    return &proto.AuthorMins{Authors: authorMins}, nil
}

func (s *server) GetAuthorByID(ctx context.Context, req *proto.IntRequest) (*proto.Author, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var author models.Author
    row := database.AuthorDB.QueryRow("SELECT name, birthdate, nationality, biography, created_at, updated_at FROM authors WHERE author_id = $1", req.RequestInt)
    err := row.Scan(&author.Name, &author.Birthdate, &author.Nationality, &author.Biography, &author.CreatedAt, &author.UpdatedAt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get author: %v", err))
        return nil, fmt.Errorf("failed to get author: %v", err)
    }

    return &proto.Author{
    	Name:        author.Name,
    	Birthdate:   author.Birthdate.Format("2006-01-02"),
    	Nationality: *author.Nationality,
    	Biography:   *author.Biography,
    	CreatedAt:   author.CreatedAt.Format(time.RFC3339),
    	UpdatedAt:   author.UpdatedAt.Format(time.RFC3339),
    }, nil
}

func (s *server) EditAuthor(ctx context.Context, req *proto.UpdateAuthor) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    _, err := time.Parse("2006-01-02", req.NewBirthdate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse birthdate: %v", err))
        return nil, fmt.Errorf("failed to parse birthdate: %v", err)
    }
    author := models.UpdateAuthor{
    	AuthorID:       int(req.AuthorId),
    	NewName:        req.NewName,
    	NewBirthdate:   req.NewBirthdate,
    	NewNationality: req.NewNationality,
    	NewBiography:   req.NewBiography,
    	UpdatedAt:      time.Now(),
    }

    // validate
    if err := ModelValidator(author); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to validate author: %v", err))
        return nil, fmt.Errorf("failed to validate author: %v", err)
    }

    // check if id exists, and fetch created_at
    row := database.AuthorDB.QueryRow("SELECT created_at FROM authors WHERE author_id = $1", author.AuthorID)
    var storedCreatedAt string
    err = row.Scan(&storedCreatedAt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] author id is invalid: %v", err))
        return nil, fmt.Errorf("author id is invalid: %v", err)
    }

    // update author
    tx, err := database.AuthorDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [AuthorDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [AuthorDB]", err)
    }

    _, err = tx.Exec("UPDATE authors SET name = $1, birthdate = $2, nationality = $3, biography = $4, updated_at = $5 WHERE author_id = $6",
        author.NewName, author.NewBirthdate, author.NewNationality, author.NewBiography, author.UpdatedAt, author.AuthorID)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to update author: %v", err))
        return nil, fmt.Errorf("failed to update author: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: fmt.Sprintf("author %s updated at %s", author.NewName, author.UpdatedAt.Format("2006-01-02 15:04:05"))}, nil
}

func (s *server) DeleteAuthor(ctx context.Context, req *proto.IntRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    // check if id exists
    var scan int
    err := database.AuthorDB.QueryRow("SELECT 1 AS exists FROM authors WHERE author_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] author id is invalid")
        return nil, fmt.Errorf("author id is invalid")
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author existence: %v", err))
        return nil, fmt.Errorf("failed to check author existence: %v", err)
    }

    // check if author is in use by books, inter-service call to bookservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    bookServiceClient := proto.NewBookAndBorrowServiceClient(interServiceConn)
    isAuthorInUse, err := bookServiceClient.IsAuthorInUseByBook(outCtx, &proto.IntRequest{RequestInt: req.RequestInt})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author in use: %v", err))
        return nil, fmt.Errorf("failed to check author in use: %v", err)
    }
    if isAuthorInUse.ResponseBool {
        logger.LogThis("[ERROR] author is in use by books")
        return nil, fmt.Errorf("author is in use by books")
    }

    // OK delete
    _, err = database.AuthorDB.Exec("DELETE FROM authors WHERE author_id = $1", req.RequestInt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to delete author: %v", err))
        return nil, fmt.Errorf("failed to delete author: %v", err)
    }

    return &proto.StringResponse{ResponseStr: fmt.Sprintf("author %d deleted", req.RequestInt)}, nil
}

func (s *server) DoesAuthorExist(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var scan int
    err := database.AuthorDB.QueryRow("SELECT 1 AS exists FROM authors WHERE author_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        // debug: check 'quick patch' on REST interface
        return &proto.BoolResponse{ResponseBool: false}, nil
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author existence: %v", err))
        return nil, fmt.Errorf("failed to check author existence: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: true}, nil
}

func (s *server) CreateCategory(ctx context.Context, req *proto.Category) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    category := models.Category{
    	Name:        "",
    	Description: new(string),
    	CreatedAt:   time.Time{},
    	UpdatedAt:   time.Time{},
    }

    category.Name = req.Name    
    category.Description = new(string)
    *category.Description = req.Description
    category.CreatedAt = time.Now()
    category.UpdatedAt = time.Now()

    // check each field
    if category.Name == "" {
        logger.LogThis("[ERROR] name is required")
        return nil, fmt.Errorf("name is required")
    }

    // check if category already exists [Duplicate Entry]
    row := database.CategoryDB.QueryRow("SELECT 1 AS exists FROM categories WHERE name = $1", category.Name)
    var exists bool
    err := row.Scan(&exists)
    if err == nil && exists {
        logger.LogThis("[ERROR] category already exists [Duplicate Entry]")
        return nil, fmt.Errorf("category already exists [Duplicate Entry]")
    } else if err == nil && !exists {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check if category already exists [Duplicate Entry]: %v", err))
        return nil, fmt.Errorf("failed to check if category already exists [Duplicate Entry]: %v", err)
    }

    tx, err := database.CategoryDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [CategoryDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [CategoryDB]", err)
    }

    _, err = tx.Exec("INSERT INTO categories (name, description, created_at, updated_at) VALUES ($1, $2, $3, $4)", category.Name, category.Description, category.CreatedAt, category.UpdatedAt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to create category: %v", err))
        return nil, fmt.Errorf("failed to create category: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "Category created successfully"}, nil
}

func (s *server) GetCategories(ctx context.Context, req *proto.IDLimits) (*proto.CategoryMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var categorymins []*proto.CategoryMin
    rows, err := database.CategoryDB.Query("SELECT category_id, name FROM categories WHERE category_id BETWEEN $1 AND $2", req.Min, req.Max)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get categories: %v", err))
        return nil, fmt.Errorf("failed to get categories: %v", err)
    }
    defer rows.Close()

    for rows.Next() {
        var categoryID int32
        var name string
        if err := rows.Scan(&categoryID, &name); err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan category: %v", err))
            return nil, fmt.Errorf("failed to scan category: %v", err)
        }
        categorymins = append(categorymins, &proto.CategoryMin{
            CategoryId: categoryID,
            Name:       name,
        })
    }

    return &proto.CategoryMins{Categories: categorymins}, nil
}

func (s *server) GetCategoryByID(ctx context.Context, req *proto.IntRequest) (*proto.Category, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var category models.Category
    row := database.CategoryDB.QueryRow("SELECT name, description, created_at, updated_at FROM categories WHERE category_id = $1", req.RequestInt)
    err := row.Scan(&category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get category: %v", err))
        return nil, fmt.Errorf("failed to get category: %v", err)
    }

    return &proto.Category{
        Name:       category.Name,
        Description: *category.Description,
        CreatedAt:    category.CreatedAt.Format(time.RFC3339),
        UpdatedAt:    category.UpdatedAt.Format(time.RFC3339),
    }, nil
}

func (s *server) GetCategoriesByName(ctx context.Context, req *proto.StringRequest) (*proto.CategoryMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var categoryMins []*proto.CategoryMin

    rows, err := database.CategoryDB.Query("SELECT category_id, name FROM categories WHERE name ILIKE $1", "%"+req.RequestStr+"%")
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get categories: %v", err))
        return nil, fmt.Errorf("failed to get categories: %v", err)
    }
    defer rows.Close()

    for rows.Next() {
        var categoryMin proto.CategoryMin
        err := rows.Scan(&categoryMin.CategoryId, &categoryMin.Name)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan category: %v", err))
            return nil, fmt.Errorf("failed to scan category: %v", err)
        }
        categoryMins = append(categoryMins, &categoryMin)
    }

    return &proto.CategoryMins{Categories: categoryMins}, nil
}

func (s *server) EditCategory(ctx context.Context, req *proto.UpdateCategory) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    category := models.UpdateCategory{
    	CategoryID:     int(req.CategoryId),
    	NewName:        req.NewName,
    	NewDescription: req.NewDescription,
    	UpdatedAt:      time.Now(),
    }

    // validate
    if err := ModelValidator(category); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to validate category: %v", err))
        return nil, fmt.Errorf("failed to validate category: %v", err)
    }

    // check if id exists, and fetch created_at
    row := database.CategoryDB.QueryRow("SELECT created_at FROM categories WHERE category_id = $1", category.CategoryID)
    var storedCreatedAt string
    err := row.Scan(&storedCreatedAt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] category id is invalid: %v", err))
        return nil, fmt.Errorf("category id is invalid: %v", err)
    }

    tx, err := database.CategoryDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [CategoryDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [CategoryDB]", err)
    }

    _, err = tx.Exec("UPDATE categories SET name = $1, description = $2, updated_at = $3 WHERE category_id = $4", category.NewName, category.NewDescription, category.UpdatedAt, category.CategoryID)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to update category: %v", err))
        return nil, fmt.Errorf("failed to update category: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{
        ResponseStr: fmt.Sprintf("category %d updated successfully to %s", category.CategoryID, category.NewName),
    }, nil
}

func (s *server) DeleteCategory(ctx context.Context, req *proto.IntRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    
    // check if category id exists
    var scan int
    err := database.CategoryDB.QueryRow("SELECT 1 FROM categories WHERE category_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] category id is unavailable")
        return nil, fmt.Errorf("category id is unavailable")
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category existence: %v", err))
        return nil, fmt.Errorf("failed to check category existence: %v", err)
    }

    // check if category is in use, inter-service call to bookservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    var isCategoryInUse *proto.BoolResponse
    bookServiceClient := proto.NewBookAndBorrowServiceClient(interServiceConn)
    isCategoryInUse, err = bookServiceClient.IsCategoryInUseByBook(outCtx, &proto.IntRequest{RequestInt: req.RequestInt})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category in use: %v", err))
        return nil, fmt.Errorf("failed to check category in use: %v", err)
    }
    if isCategoryInUse.ResponseBool {
        logger.LogThis("[ERROR] category is in use")
        return nil, fmt.Errorf("category is in use")
    }

    // OK delete
    _, err = database.CategoryDB.Exec("DELETE FROM categories WHERE category_id = $1", req.RequestInt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to delete category: %v", err))
        return nil, fmt.Errorf("failed to delete category: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "Category deleted successfully"}, nil
}

func (s *server) DoesCategoryExist(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var exists bool
    err := database.CategoryDB.QueryRow("SELECT 1 AS exists FROM categories WHERE category_id = $1 LIMIT 1", req.RequestInt).Scan(&exists)
    if err == sql.ErrNoRows {
        return &proto.BoolResponse{ResponseBool: false}, nil
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category existence: %v", err))
        return nil, fmt.Errorf("failed to check category existence: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: true}, nil
}

func (s *server) IsAuthorInUseByBook(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var scan int
    err := database.BookDB.QueryRow("SELECT 1 AS exists FROM books WHERE author_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        return &proto.BoolResponse{ResponseBool: false}, nil
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author in use: %v", err))
        return nil, fmt.Errorf("failed to check author in use: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: true}, nil
}

func (s *server) IsCategoryInUseByBook(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var scan int
    err := database.BookDB.QueryRow("SELECT 1 AS exists FROM books WHERE category_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        return &proto.BoolResponse{ResponseBool: false}, nil
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category in use: %v", err))
        return nil, fmt.Errorf("failed to check category in use: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: true}, nil
}

func (s *server) CreateBook(ctx context.Context, req *proto.Book) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    parsedPublishedDate, err := time.Parse("2006-01-02", req.PublishedDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse published date: %v", err))
        return nil, fmt.Errorf("failed to parse published date: %v", err)
    }
    book := models.Book{
    	Title:          req.Title,
    	CategoryID:     int(req.CategoryId),
    	AuthorID:       int(req.AuthorId),
    	PublishedDate:  &parsedPublishedDate,
    	ISBN:           &req.Isbn,
    	TotalStock:     int(req.TotalStock),
    	AvailableStock: int(req.AvailableStock),
    	CreatedAt:      time.Now(),
    	UpdatedAt:      time.Now(),
    }

    // validate
    if err := ModelValidator(book); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] title, category_id, author_id, published_date, isbn, total_stock, available_stock are required [Insufficient Input]: %v", err))
        return nil, fmt.Errorf("title, category_id, author_id, published_date, isbn, total_stock, available_stock are required [Insufficient Input]: %v", err)
    }

    // check if category id exists, inter-service call to categoryservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    categoryServiceClient := proto.NewCategoryServiceClient(interServiceConn)
    doesCategoryExist, err := categoryServiceClient.DoesCategoryExist(outCtx, &proto.IntRequest{RequestInt: int32(book.CategoryID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category existence: %v", err))
        return nil, fmt.Errorf("failed to check category existence: %v", err)
    }

    if !doesCategoryExist.ResponseBool {
        logger.LogThis("[ERROR] category id is unavailable")
        return nil, fmt.Errorf("category id is unavailable")
    }

    // check if author id exists, inter-service call to authorservice
    authorServiceClient := proto.NewAuthorServiceClient(interServiceConn)
    doesAuthorExist, err := authorServiceClient.DoesAuthorExist(outCtx, &proto.IntRequest{RequestInt: int32(book.AuthorID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author existence: %v", err))
        return nil, fmt.Errorf("failed to check author existence: %v", err)
    }

    if !doesAuthorExist.ResponseBool {
        logger.LogThis("[ERROR] author id is unavailable")
        return nil, fmt.Errorf("author id is unavailable")
    }

    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to start transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to start transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("INSERT INTO books (title, category_id, author_id, published_date, isbn, total_stock, available_stock, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
        book.Title, book.CategoryID, book.AuthorID, book.PublishedDate, book.ISBN, book.TotalStock, book.AvailableStock, book.CreatedAt, book.UpdatedAt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to insert book: %v", err))
        return nil, fmt.Errorf("failed to insert book: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: fmt.Sprintf("Book %s created", book.Title)}, nil
}

func (s *server) GetBooks(ctx context.Context, req *proto.IDLimits) (*proto.BookMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var bookMins []*proto.BookMin

    rows, err := database.BookDB.Query("SELECT book_id, title, category_id, author_id, published_date, available_stock FROM books WHERE book_id BETWEEN $1 AND $2", req.Min, req.Max)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get books: %v", err))
        return nil, fmt.Errorf("failed to get books: %v", err)
    }

    for rows.Next() {
        var bookMin proto.BookMin
        err := rows.Scan(&bookMin.BookId, &bookMin.Title, &bookMin.CategoryId, &bookMin.AuthorId, &bookMin.PublishedDate, &bookMin.AvailableStock)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan book: %v", err))
            return nil, fmt.Errorf("failed to scan book: %v", err)
        }
        bookMins = append(bookMins, &bookMin)
    }

    return &proto.BookMins{Books: bookMins}, nil
}

func (s *server) GetBooksByDate(ctx context.Context, req *proto.DateLimits) (*proto.BookMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var bookMins []*proto.BookMin

    // validate input date format
    _, err := time.Parse("2006-01-02", req.StartDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse min date: %v", err))
        return nil, fmt.Errorf("failed to parse min date: %v", err)
    }
    _, err = time.Parse("2006-01-02", req.EndDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse max date: %v", err))
        return nil, fmt.Errorf("failed to parse max date: %v", err)
    }

    rows, err := database.BookDB.Query("SELECT book_id, title, category_id, author_id, published_date, available_stock FROM books WHERE published_date BETWEEN $1 AND $2", req.StartDate, req.EndDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get books: %v", err))
        return nil, fmt.Errorf("failed to get books: %v", err)
    }

    for rows.Next() {
        var bookMin proto.BookMin
        err := rows.Scan(&bookMin.BookId, &bookMin.Title, &bookMin.CategoryId, &bookMin.AuthorId, &bookMin.PublishedDate, &bookMin.AvailableStock)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan book: %v", err))
            return nil, fmt.Errorf("failed to scan book: %v", err)
        }
        bookMins = append(bookMins, &bookMin)
    }

    return &proto.BookMins{Books: bookMins}, nil
}

func (s *server) GetBooksByName(ctx context.Context, req *proto.StringRequest) (*proto.BookMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var bookMins []*proto.BookMin

    // ilike %str%
    rows, err := database.BookDB.Query("SELECT book_id, title, category_id, author_id, published_date, available_stock FROM books WHERE title ILIKE $1", "%"+req.RequestStr+"%")
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get books: %v", err))
        return nil, fmt.Errorf("failed to get books: %v", err)
    }

    for rows.Next() {
        var bookMin proto.BookMin
        err := rows.Scan(&bookMin.BookId, &bookMin.Title, &bookMin.CategoryId, &bookMin.AuthorId, &bookMin.PublishedDate, &bookMin.AvailableStock)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan book: %v", err))
            return nil, fmt.Errorf("failed to scan book: %v", err)
        }
        bookMins = append(bookMins, &bookMin)
    }

    return &proto.BookMins{Books: bookMins}, nil
}

func (s *server) GetBookByID(ctx context.Context, req *proto.IntRequest) (*proto.Book, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var book models.Book
    row := database.BookDB.QueryRow("SELECT title, category_id, author_id, published_date, isbn, total_stock, available_stock, created_at, updated_at FROM books WHERE book_id = $1", req.RequestInt)

    err := row.Scan(&book.Title, &book.CategoryID, &book.AuthorID, &book.PublishedDate, &book.ISBN, &book.TotalStock, &book.AvailableStock, &book.CreatedAt, &book.UpdatedAt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get book: %v", err))
        return nil, fmt.Errorf("failed to get book: %v", err)
    }

    return &proto.Book{
        Title:          book.Title,
        CategoryId:     int32(book.CategoryID),
        AuthorId:       int32(book.AuthorID),
        PublishedDate:  book.PublishedDate.Format("2006-01-02"),
        Isbn:           *book.ISBN,
        TotalStock:     int32(book.TotalStock),
        AvailableStock: int32(book.AvailableStock),
        CreatedAt:      book.CreatedAt.Format("2006-01-02"),
        UpdatedAt:      book.UpdatedAt.Format("2006-01-02"),
    }, nil
}

func (s *server) EditBook(ctx context.Context, req *proto.UpdateBook) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    _, err := time.Parse("2006-01-02", req.NewPublishedDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse published date: %v", err))
        return nil, fmt.Errorf("failed to parse published date: %v", err)
    }

    book := models.UpdateBook{
    	BookID:            int(req.BookId),
    	NewTitle:          req.NewTitle,
    	NewCategoryID:     int(req.NewCategoryId),
    	NewAuthorID:       int(req.NewAuthorId),
    	NewPublishedDate:  req.NewPublishedDate,
    	NewISBN:           req.NewIsbn,
    	NewTotalStock:     int(req.NewTotalStock),
    	NewAvailableStock: int(req.NewAvailableStock),
    	UpdatedAt:         time.Now(),
    }

    // validate model
    if err := ModelValidator(book); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] title, category_id, author_id, published_date, isbn, total_stock, available_stock are required [Insufficient Input]: %v", err))
        return nil, fmt.Errorf("title, category_id, author_id, published_date, isbn, total_stock, available_stock are required [Insufficient Input]: %v", err)
    }

    // check if category id exists, inter-service call to categoryservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    categoryServiceClient := proto.NewCategoryServiceClient(interServiceConn)
    doesCategoryExist, err := categoryServiceClient.DoesCategoryExist(outCtx, &proto.IntRequest{RequestInt: int32(book.NewCategoryID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category: %v", err))
        return nil, fmt.Errorf("failed to check category: %v", err)
    }
    if !doesCategoryExist.ResponseBool {
        logger.LogThis("[ERROR] category does not exist")
        return nil, fmt.Errorf("category does not exist")
    }

    // check if author id exists, inter-service call to authorservice
    authorServiceClient := proto.NewAuthorServiceClient(interServiceConn)
    doesAuthorExist, err := authorServiceClient.DoesAuthorExist(outCtx, &proto.IntRequest{RequestInt: int32(book.NewAuthorID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check author: %v", err))
        return nil, fmt.Errorf("failed to check author: %v", err)
    }
    if !doesAuthorExist.ResponseBool {
        logger.LogThis("[ERROR] author does not exist")
        return nil, fmt.Errorf("author does not exist")
    }

    // check if book id exists, inter-service call to bookservice
    var scan int
    err = database.BookDB.QueryRow("SELECT 1 FROM books WHERE book_id = $1", book.BookID).Scan(&scan)
    if err != nil && err == sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] book does not exist: %v", err))
        return nil, fmt.Errorf("book does not exist: %v", err)
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check book: %v", err))
        return nil, fmt.Errorf("failed to check book: %v", err)
    }

    // update book
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("UPDATE books SET title = $1, category_id = $2, author_id = $3, published_date = $4, isbn = $5, total_stock = $6, available_stock = $7, updated_at = $8 WHERE book_id = $9",
        book.NewTitle, book.NewCategoryID, book.NewAuthorID, book.NewPublishedDate, book.NewISBN, book.NewTotalStock, book.NewAvailableStock, book.UpdatedAt, book.BookID)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to update book: %v", err))
        return nil, fmt.Errorf("failed to update book: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully updated"}, nil
}

func (s *server) DeleteBook(ctx context.Context, req *proto.IntRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    // check if book exists
    var scan int
    err := database.BookDB.QueryRow("SELECT 1 AS exists FROM books WHERE book_id = $1 LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] book does not exist")
        return nil, fmt.Errorf("book does not exist")
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check book existence: %v", err))
        return nil, fmt.Errorf("failed to check book existence: %v", err)
    }

    // delete book
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("DELETE FROM books WHERE book_id = $1", req.RequestInt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to delete book: %v", err))
        return nil, fmt.Errorf("failed to delete book: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully deleted"}, nil
}

func (s *server) DoesUserStillBorrow(ctx context.Context, req *proto.IntRequest) (*proto.BoolResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var scan int
    err := database.BookDB.QueryRow("SELECT 1 AS exists FROM borrowing WHERE user_id = $1 AND returned = 'f' LIMIT 1", req.RequestInt).Scan(&scan)
    if err == nil { // NGs
        return &proto.BoolResponse{ResponseBool: true}, nil
    } else if err != sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check borrow: %v", err))
        return nil, fmt.Errorf("failed to check borrow: %v", err)
    }

    return &proto.BoolResponse{ResponseBool: false}, nil
}

func (s *server) CreateBorrow(ctx context.Context, req *proto.Borrow) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    parsedReturnDate, err := time.Parse("2006-01-02", req.ReturnDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse return date: %v", err))
        return nil, fmt.Errorf("failed to parse return date: %v", err)
    }

    borrow := models.Borrow{
    	BookID:       int(req.BookId),
    	UserID:       int(req.UserId),
    	BorrowedDate: time.Now(),
    	ReturnDate:   parsedReturnDate,
        ReturnedDate: nil,
    	Returned:     false,
    }

    // validate model
    if err := ModelValidator(borrow); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] book_id, user_id, borrowed_date, return_date are required [Insufficient Input]: %v", err))
        return nil, fmt.Errorf("book_id, user_id, borrowed_date, return_date are required [Insufficient Input]: %v", err)
    }
    
    // check if user exists, inter-service call to userservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    userServiceClient := proto.NewUserServiceClient(interServiceConn)
    doesUserExist, err := userServiceClient.DoesUserExist(outCtx, &proto.IntRequest{RequestInt: int32(borrow.UserID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check user: %v", err))
        return nil, fmt.Errorf("failed to check user: %v", err)
    }
    if !doesUserExist.ResponseBool {
        logger.LogThis("[ERROR] user does not exist")
        return nil, fmt.Errorf("user does not exist")
    }

    // check if book available
    var scan int
    err = database.BookDB.QueryRow("SELECT 1 AS exists FROM books WHERE book_id = $1 AND available_stock > 0 LIMIT 1", borrow.BookID).Scan(&scan)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] book is not available")
        return nil, fmt.Errorf("book is not available")
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check book existence: %v", err))
        return nil, fmt.Errorf("failed to check book existence: %v", err)
    }

    // OK create borrow
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("INSERT INTO borrowing (book_id, user_id, borrowed_date, return_date, returned_date, returned) VALUES ($1, $2, $3, $4, $5, $6)", borrow.BookID, borrow.UserID, borrow.BorrowedDate, borrow.ReturnDate, borrow.ReturnedDate, borrow.Returned)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to create borrow: %v", err))
        return nil, fmt.Errorf("failed to create borrow: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully borrowed"}, nil
}

func (s *server) CreateReturn(ctx context.Context, req *proto.IntRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    // check if borrow exists
    var scan int
    err := database.BookDB.QueryRow("SELECT 1 AS exists FROM borrowing WHERE borrowing_id = $1 AND returned = 'f' LIMIT 1", req.RequestInt).Scan(&scan)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] borrow does not exist")
        return nil, fmt.Errorf("borrow does not exist")
    } else if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check borrow existence: %v", err))
        return nil, fmt.Errorf("failed to check borrow existence: %v", err)
    }

    // OK create return
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("UPDATE borrowing SET returned = 't', returned_date = $1 WHERE borrowing_id = $2 AND returned = 'f'", time.Now(), req.RequestInt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to create return: %v", err))
        return nil, fmt.Errorf("failed to create return: %v", err)
    }
    
    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully returned"}, nil
}

func (s *server) GetBorrowings(ctx context.Context, req *proto.IDLimits) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 'f' AND borrowing_id BETWEEN $1 AND $2", req.Min, req.Max)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetBorrowingsByDate(ctx context.Context, req *proto.DateLimits) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 'f' AND borrowed_date BETWEEN $1 AND $2", req.StartDate, req.EndDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetBorrowingsByUserID(ctx context.Context, req *proto.IntRequest) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    // check if user exist, inter-service call to userservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    userServiceClient := proto.NewUserServiceClient(interServiceConn)
    doesUserExist, err := userServiceClient.DoesUserExist(outCtx, &proto.IntRequest{RequestInt: req.RequestInt})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check user: %v", err))
        return nil, fmt.Errorf("failed to check user: %v", err)
    }
    if !doesUserExist.ResponseBool {
        logger.LogThis("[ERROR] user does not exist")
        return nil, fmt.Errorf("user does not exist")
    }

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 'f' AND user_id = $1", req.RequestInt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetReturns(ctx context.Context, req *proto.IDLimits) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 't' AND borrowing_id BETWEEN $1 AND $2", req.Min, req.Max)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetReturnsByDate(ctx context.Context, req *proto.DateLimits) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 't' AND borrowed_date BETWEEN $1 AND $2", req.StartDate, req.EndDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetReturnsByUserID(ctx context.Context, req *proto.IntRequest) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    // check if user exist, inter-service call to userservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    userServiceClient := proto.NewUserServiceClient(interServiceConn)
    doesUserExist, err := userServiceClient.DoesUserExist(outCtx, &proto.IntRequest{RequestInt: req.RequestInt})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check user: %v", err))
        return nil, fmt.Errorf("failed to check user: %v", err)
    }
    if !doesUserExist.ResponseBool {
        logger.LogThis("[ERROR] user does not exist")
        return nil, fmt.Errorf("user does not exist")
    }

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 't' AND user_id = $1", req.RequestInt)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) GetOverdues(ctx context.Context, req *proto.DateLimits) (*proto.BorrowOrReturnMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var borrowOrReturnMins []*proto.BorrowOrReturnMin

    rows, err := database.BookDB.Query("SELECT borrowing_id, book_id, user_id, borrowed_date FROM borrowing WHERE returned = 'f' AND return_date < borrowed_date")
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to get borrowings: %v", err))
        return nil, fmt.Errorf("failed to get borrowings: %v", err)
    }

    for rows.Next() {
        var borrowOrReturnMin proto.BorrowOrReturnMin
        err := rows.Scan(&borrowOrReturnMin.BorrowingId, &borrowOrReturnMin.BookId, &borrowOrReturnMin.UserId, &borrowOrReturnMin.BorrowedDate)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan borrowings: %v", err))
            return nil, fmt.Errorf("failed to scan borrowings: %v", err)
        }

        borrowOrReturnMins = append(borrowOrReturnMins, &borrowOrReturnMin)
    }

    return &proto.BorrowOrReturnMins{Message: "success", Borrowings: borrowOrReturnMins}, nil
}

func (s *server) EditBorrow(ctx context.Context, req *proto.UpdateBorrow) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }


    // validate dates
    _, err := time.Parse("2006-01-02", req.NewBorrowedDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse date: %v", err))
        return nil, fmt.Errorf("failed to parse date: %v", err)
    }
    _, err = time.Parse("2006-01-02", req.NewReturnDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse date: %v", err))
        return nil, fmt.Errorf("failed to parse date: %v", err)
    }
    _, err = time.Parse("2006-01-02", req.NewReturnedDate)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to parse date: %v", err))
        return nil, fmt.Errorf("failed to parse date: %v", err)
    }
    
    updateBorrow := models.UpdateBorrow{
    	BorrowingID:     int(req.BorrowingId),
    	NewBookID:       int(req.NewBookId),
    	NewUserID:       int(req.NewUserId),
    	NewBorrowedDate: req.NewBorrowedDate,
    	NewReturnDate:   req.NewReturnDate,
    	NewReturnedDate:   req.NewReturnedDate,
    	NewReturned:     req.NewReturned,
    }

    // validate model
    if err := ModelValidator(updateBorrow); err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] borrowing_id, new_book_id, new_user_id, new_borrowed_date, new_return_date, new_returned_date, new_returned are required [Insufficient Input]: %v", err))
        return nil, fmt.Errorf("borrowing_id, new_book_id, new_user_id, new_borrowed_date, new_return_date, new_returned_date, new_returned are required [Insufficient Input]: %v", err)
    }

    // check if borrow_id exists
    var scan int
    err = database.BookDB.QueryRow("SELECT 1 FROM borrowing WHERE borrowing_id = $1 LIMIT 1", req.BorrowingId).Scan(&scan)
    if err != nil && err == sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] borrow_id does not exist: %v", err))
        return nil, fmt.Errorf("failed to check if borrow_id exists: %v", err)
    } else if err != sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check if borrow_id exists: %v", err))
    }

    // check if book exists
    err = database.BookDB.QueryRow("SELECT 1 FROM books WHERE book_id = $1 LIMIT 1", req.NewBookId).Scan(&scan)
    if err != nil && err == sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] book does not exist: %v", err))
        return nil, fmt.Errorf("failed to check if book exists: %v", err)
    } else if err != sql.ErrNoRows {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check if book exists: %v", err))
    }

    // check if user exists, inter-service call to userservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    userServiceClient := proto.NewUserServiceClient(interServiceConn)
    doesUserExist, err := userServiceClient.DoesUserExist(outCtx, &proto.IntRequest{RequestInt: int32(updateBorrow.NewUserID)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check user: %v", err))
        return nil, fmt.Errorf("failed to check user: %v", err)
    }
    if !doesUserExist.ResponseBool {
        logger.LogThis("[ERROR] user does not exist")
        return nil, fmt.Errorf("user does not exist")
    }

    // update borrow
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("UPDATE borrowing SET book_id = $1, user_id = $2, borrowed_date = $3, return_date = $4, returned = $5 WHERE borrowing_id = $6", req.NewBookId, req.NewUserId, req.NewBorrowedDate, req.NewReturnDate, req.NewReturned, req.BorrowingId)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to update borrow: %v", err))
        return nil, fmt.Errorf("failed to update borrow: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully updated borrow"}, nil
}

func (s *server) DeleteBorrow(ctx context.Context, req *proto.IntRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    // check if borrowing exists, and get book_id
    var bookId int
    err := database.BookDB.QueryRow("SELECT book_id FROM borrowing WHERE borrowing_id = $1 LIMIT 1", req.RequestInt).Scan(&bookId)
    if err == sql.ErrNoRows {
        logger.LogThis("[ERROR] borrowing not found")
        return nil, fmt.Errorf("borrowing not found")
    }
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to fetch borrowing details: %v", err))
        return nil, fmt.Errorf("failed to fetch borrowing details: %v", err)
    }

    // OK
    tx, err := database.BookDB.Begin()
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to begin transaction: %v [BookDB]", err))
        return nil, fmt.Errorf("failed to begin transaction: %v [BookDB]", err)
    }

    _, err = tx.Exec("DELETE FROM borrowing WHERE borrowing_id = $1", req.RequestInt)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to delete borrow: %v", err))
        return nil, fmt.Errorf("failed to delete borrow: %v", err)
    }

    // extra: return the book
    _, err = tx.Exec("UPDATE books SET available_stock = available_stock + 1 WHERE book_id = $1", bookId)
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to return book: %v", err))
        return nil, fmt.Errorf("failed to return book: %v", err)
    }

    err = tx.Commit()
    if err != nil {
        tx.Rollback()
        logger.LogThis(fmt.Sprintf("[ERROR] failed to commit transaction: %v", err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    return &proto.StringResponse{ResponseStr: "successfully deleted borrow"}, nil
}

func (s *server) GetBookRecommendations(ctx context.Context, req *proto.GetRecommendation) (*proto.BookMins, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    var bookMins []*proto.BookMin

    // check if category_id exists, inter-service call to categoryservice
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        logger.LogThis("[ERROR] failed to get metadata")
        return nil, fmt.Errorf("failed to get metadata")
    }
    outCtx := metadata.NewOutgoingContext(ctx, md)
    categoryServiceClient := proto.NewCategoryServiceClient(interServiceConn)
    doesCategoryExist, err := categoryServiceClient.DoesCategoryExist(outCtx, &proto.IntRequest{RequestInt: int32(req.CategoryId)})
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to check category: %v", err))
        return nil, fmt.Errorf("failed to check category: %v", err)
    }
    if !doesCategoryExist.ResponseBool {
        logger.LogThis("[ERROR] category does not exist")
        return nil, fmt.Errorf("category does not exist")
    }

    // debug limit
    // fmt.Println(req.Limit)

    rows, err := database.BookDB.Query("SELECT book_id, title FROM books WHERE category_id = $1 ORDER BY RANDOM() LIMIT $2", req.CategoryId, req.Limit)
    if err != nil {
        logger.LogThis(fmt.Sprintf("[ERROR] failed to fetch books: %v", err))
        return nil, fmt.Errorf("failed to fetch books: %v", err)
    }

    for rows.Next() {
        var bookMin proto.BookMin
        err := rows.Scan(&bookMin.BookId, &bookMin.Title)
        if err != nil {
            logger.LogThis(fmt.Sprintf("[ERROR] failed to scan book: %v", err))
            return nil, fmt.Errorf("failed to scan book: %v", err)
        }
        bookMins = append(bookMins, &bookMin)
    }

    return &proto.BookMins{Books: bookMins}, nil
}

func (s *server) HelloWorld(ctx context.Context, req *proto.StringRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    username, err := validateJWT(ctx)
    if err != nil {
        return nil, err
    }
    
    message := req.RequestStr + ", " + username
    return &proto.StringResponse{ResponseStr: message}, nil
}

func (s *server) Ping(ctx context.Context, req *emptypb.Empty) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    message := "Pong"
    return &proto.StringResponse{ResponseStr: message}, nil
}

func (s *server) AuthWithoutCredentials(ctx context.Context, req *proto.StringRequest) (*proto.StringResponse, error) {
    if _, err := validateJWT(ctx); err != nil {
        return nil, err
    }

    token, err := jwtgenerator.GenerateJWT(req.RequestStr)
    if err != nil {
        return nil, err
    }
    return &proto.StringResponse{ResponseStr: token}, nil
}

func main() {
    // err := godotenv.Load()
    // if err != nil {
    //     log.Fatal("Error loading .env file, trying to get from os")
    // }

    database.ConnectBookDB()
    database.ConnectAuthorDB()
    database.ConnectUserDB()
    database.ConnectCategoryDB()

    // gRPC setup
    lis, err := net.Listen("tcp", "0.0.0.0:50051")
    if err != nil {
        logger.LogThis(fmt.Sprintf("[FATAL] failed to listen: %v", err))
    }
    s := grpc.NewServer()

    proto.RegisterUtilServiceServer(s, &server{})
    proto.RegisterUserServiceServer(s, &server{})
    proto.RegisterAuthorServiceServer(s, &server{})
    proto.RegisterCategoryServiceServer(s, &server{})
    proto.RegisterBookAndBorrowServiceServer(s, &server{})

    reflection.Register(s)

    // goroutine gRPC
    go func() {
        logger.LogThis("[INFO] Server is running on port 50051...")
        if err := s.Serve(lis); err != nil {
            logger.LogThis(fmt.Sprintf("[FATAL] failed to serve: %v", err))
        }
    }()

    // init inter-service connection
    InitGRPCConnection()
    defer CloseGRPCConnection()

    // stopper
    select {}
}
