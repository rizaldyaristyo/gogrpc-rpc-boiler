package main

import (
	"context"
	"log"
	"strconv"
	"time"

	proto "gogrpc-rpc-boiler/proto"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
    app := fiber.New()

    // gRPC setup
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    // gRPC clients
    utilClient := proto.NewUtilServiceClient(conn)
    userClient := proto.NewUserServiceClient(conn)
    authorClient := proto.NewAuthorServiceClient(conn)
    categoryClient := proto.NewCategoryServiceClient(conn)
    bookClient := proto.NewBookAndBorrowServiceClient(conn)

    app.Get("/hello/:name", func(c *fiber.Ctx) error {
        name := c.Params("name")
        bearerToken := c.Get("Authorization")
        if bearerToken == "" {
            return c.Status(401).JSON(fiber.Map{"error": "Authorization token missing"})
        }
        // add token to metadata
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // rpc call
        req := &proto.StringRequest{RequestStr: name}
        res, err := utilClient.HelloWorld(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UtilService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Get("/ping", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        res, err := utilClient.Ping(ctx, &emptypb.Empty{})
        if err != nil {
            return c.Status(500).SendString("Error calling PingService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Get("/authwithoutcredentials/:username", func(c *fiber.Ctx) error {
        username := c.Params("username")

        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.StringRequest{RequestStr: username}
        res, err := utilClient.AuthWithoutCredentials(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })


    // USER REST INTERFACE

    app.Post("/createuser", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.UserSensitive{
        	Username:  c.FormValue("username"),
        	Password:  c.FormValue("password"),
        	FirstName: c.FormValue("first_name"),
        	LastName:  c.FormValue("last_name"),
        	Email:     c.FormValue("email"),
        	Role:      c.FormValue("role"),
        	CreatedAt: time.Now().Format(time.RFC3339),
        	UpdatedAt: time.Now().Format(time.RFC3339),
        }
        res, err := userClient.CreateUser(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Post("/login", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.UserPassword{Username: c.FormValue("username"), Password: c.FormValue("password")}
        res, err := userClient.LoginAuth(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Post("/changepassword", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.NewPassword{
            Username: c.FormValue("username"),
            Password: c.FormValue("password"),
            NewPassword: c.FormValue("new_password"),
        }
        res, err := userClient.ChangePassword(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Post("/deleteuser", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        intUserID, err := strconv.Atoi(c.FormValue("user_id"))
        if err != nil {
            return c.Status(500).SendString("invalid user_id")
        }

        // INPUT
        req := &proto.UserIDPassword{
            UserId: int32(intUserID),
            Password: c.FormValue("password"),
        }
        res, err := userClient.DeleteUser(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Get("/getuser/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        int_param, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("invalid user_id")
        }

        // INPUT
        req := &proto.IntRequest{RequestInt: int32(int_param)}
        res, err := userClient.GetUser(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{
            "user_id":    res.UserId,
            "username":   res.Username,
            "first_name": res.FirstName,
            "last_name":  res.LastName,
            "email":      res.Email,
            "role":       res.Role,
        })
    })

    app.Get("doesuserexist/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        int_param, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("invalid user_id")
        }

        // INPUT
        req := &proto.IntRequest{RequestInt: int32(int_param)}
        res, err := userClient.DoesUserExist(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling UserService: " + err.Error())
        }

        return c.JSON(fiber.Map{"exists": res.ResponseBool})
    })
    
    // AUTHOR REST INTERFACE
    app.Post("/createauthor", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.Author{
        	Name:        c.FormValue("name"),
        	Birthdate:   c.FormValue("birthdate"),
        	Nationality: c.FormValue("nationality"),
        	Biography:   c.FormValue("biography"),
        }

        res, err := authorClient.CreateAuthor(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(fiber.Map{"message": res.ResponseStr})
    })

    app.Post("/getauthors", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        minInt, err := strconv.Atoi(c.FormValue("min"))
        if err != nil {
            return c.Status(500).SendString("failed to convert min to int")
        }

        maxInt, err := strconv.Atoi(c.FormValue("max"))
        if err != nil {
            return c.Status(500).SendString("failed to convert max to int")
        }
        req := &proto.IDLimits{
        	Min: int32(minInt),
        	Max: int32(maxInt),
        }

        res, err := authorClient.GetAuthors(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getauthorsbyname", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.StringRequest{
        	RequestStr: c.FormValue("name"),
        }

        res, err := authorClient.GetAuthorsByName(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getauthorbyid/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        int_param, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("invalid author_id")
        }
        req := &proto.IntRequest{RequestInt: int32(int_param)}
        res, err := authorClient.GetAuthorByID(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(fiber.Map{
            "name":      res.Name,
            "birthdate": res.Birthdate,
            "nationality": res.Nationality,
            "biography": res.Biography,
            "created_at": res.CreatedAt,
            "updated_at": res.UpdatedAt,
        })
    })

    app.Post("/editauthor", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        authorIDInt, err := strconv.Atoi(c.FormValue("author_id"))
        if err != nil {
            return c.Status(500).SendString("invalid author_id")
        }
        req := &proto.UpdateAuthor{
        	AuthorId:       int32(authorIDInt),
        	NewName:        c.FormValue("new_name"),
        	NewBirthdate:   c.FormValue("new_birthdate"),
        	NewNationality: c.FormValue("new_nationality"),
        	NewBiography:   c.FormValue("new_biography"),
        }

        res, err := authorClient.EditAuthor(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/deleteauthor", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        authorIDInt, err := strconv.Atoi(c.FormValue("author_id"))
        if err != nil {
            return c.Status(500).SendString("invalid author_id")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(authorIDInt),
        }

        res, err := authorClient.DeleteAuthor(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        return c.JSON(res)
    })
        
    app.Get("/doesauthorexist/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        int_param, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("invalid author_id")
        }
        req := &proto.IntRequest{RequestInt: int32(int_param)}
        res, err := authorClient.DoesAuthorExist(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling AuthorService: " + err.Error())
        }

        // quick patch: idk what happen, when the response is false, it'd return nothing, weird
        if !res.ResponseBool {
            return c.JSON(fiber.Map{
                "response_bool": false,
            })
        } else {
            return c.JSON(res)
        }
    })

    // CATEGORY REST INTERFACE

    app.Get("/createcategory", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.Category{
        	Name:        c.FormValue("name"),
        	Description: c.FormValue("description"),
        }

        res, err := categoryClient.CreateCategory(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getcategories", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        minInt, err := strconv.Atoi(c.FormValue("min"))
        if err != nil {
            return c.Status(500).SendString("failed to convert min to int")
        }

        maxInt, err := strconv.Atoi(c.FormValue("max"))
        if err != nil {
            return c.Status(500).SendString("failed to convert max to int")
        }

        req := &proto.IDLimits{
        	Min: int32(minInt),
        	Max: int32(maxInt),
        }

        res, err := categoryClient.GetCategories(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getcategoriesbyname", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.StringRequest{
        	RequestStr: c.FormValue("name"),
        }

        res, err := categoryClient.GetCategoriesByName(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getcategorybyid/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := categoryClient.GetCategoryByID(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })


    app.Post("/editcategory", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        categoryIDInt, err := strconv.Atoi(c.FormValue("category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.UpdateCategory{
        	CategoryId:     int32(categoryIDInt),
        	NewName:        c.FormValue("new_name"),
        	NewDescription: c.FormValue("new_description"),
        }

        res, err := categoryClient.EditCategory(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })


    app.Post("/deletecategory", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        categoryIDInt, err := strconv.Atoi(c.FormValue("category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(categoryIDInt),
        }

        res, err := categoryClient.DeleteCategory(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/doescategoryexist", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        categoryIDInt, err := strconv.Atoi(c.FormValue("category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(categoryIDInt),
        }

        res, err := categoryClient.DoesCategoryExist(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling CategoryService: " + err.Error())
        }

        if !res.ResponseBool {
            return c.JSON(fiber.Map{
                "response_bool": false,
            })
        } else {
            return c.JSON(res)
        }
    })

    // BOOK AND BORROW REST INTERFACE

    app.Get("/isauthorinusebybook/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.IsAuthorInUseByBook(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        if !res.ResponseBool {
            return c.JSON(fiber.Map{
                "response_bool": false,
            })
        } else {
            return c.JSON(res)
        }
    })

    app.Get("/iscategoryinusebybook/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.IsCategoryInUseByBook(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        if !res.ResponseBool {
            return c.JSON(fiber.Map{
                "response_bool": false,
            })
        } else {
            return c.JSON(res)
        }
    })

    app.Post("/createbook", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        categoryIDInt, err := strconv.Atoi(c.FormValue("category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert category_id to int")
        }
        authorIDInt, err := strconv.Atoi(c.FormValue("author_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert author_id to int")
        }
        totalStockInt, err := strconv.Atoi(c.FormValue("total_stock"))
        if err != nil {
            return c.Status(500).SendString("failed to convert total_stock to int")
        }
        availableStockInt, err := strconv.Atoi(c.FormValue("available_stock"))
        if err != nil {
            return c.Status(500).SendString("failed to convert available_stock to int")
        }
        req := &proto.Book{
        		Title:          c.FormValue("title"),
        		CategoryId:     int32(categoryIDInt),
        		AuthorId:       int32(authorIDInt),
        		PublishedDate:  c.FormValue("published_date"),
        		Isbn:           c.FormValue("isbn"),
        		TotalStock:     int32(totalStockInt),
        		AvailableStock: int32(availableStockInt),
        }

        res, err := bookClient.CreateBook(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getbooks", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        minInt, err := strconv.Atoi(c.FormValue("min"))
        if err != nil {
            return c.Status(500).SendString("failed to convert min to int")
        }
        maxInt, err := strconv.Atoi(c.FormValue("max"))
        if err != nil {
            return c.Status(500).SendString("failed to convert max to int")
        }
        req := &proto.IDLimits{
        	Min: int32(minInt),
        	Max: int32(maxInt),
        }

        res, err := bookClient.GetBooks(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getbooksbydate", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        // validate start_date and end_date
        _, err := time.Parse("2006-01-02", c.FormValue("start_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse start_date: " + err.Error())
        }
        _, err = time.Parse("2006-01-02", c.FormValue("end_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse end_date: " + err.Error())
        }
        
        req := &proto.DateLimits{
        	StartDate: c.FormValue("start_date"),
        	EndDate:   c.FormValue("end_date"),
        }

        res, err := bookClient.GetBooksByDate(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getbooksbyname", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        req := &proto.StringRequest{
        	RequestStr: c.FormValue("name"),
        }

        res, err := bookClient.GetBooksByName(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getbookbyid/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.GetBookByID(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/editbook", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        bookIDInt, err := strconv.Atoi(c.FormValue("book_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert book_id to int")
        }
        categoryIDInt, err := strconv.Atoi(c.FormValue("new_category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert category_id to int")
        }
        authorIDInt, err := strconv.Atoi(c.FormValue("new_author_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert author_id to int")
        }
        totalStockInt, err := strconv.Atoi(c.FormValue("new_total_stock"))
        if err != nil {
            return c.Status(500).SendString("failed to convert total_stock to int")
        }
        availableStockInt, err := strconv.Atoi(c.FormValue("new_available_stock"))
        if err != nil {
            return c.Status(500).SendString("failed to convert available_stock to int")
        }
        req := &proto.UpdateBook{
        	BookId:            int32(bookIDInt),
        	NewTitle:          c.FormValue("new_title"),
        	NewCategoryId:     int32(categoryIDInt),
        	NewAuthorId:       int32(authorIDInt),
        	NewPublishedDate:  c.FormValue("new_published_date"),
        	NewIsbn:           c.FormValue("new_isbn"),
        	NewTotalStock:     int32(totalStockInt),
        	NewAvailableStock: int32(availableStockInt),
        }

        res, err := bookClient.EditBook(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/deletebook", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.FormValue("book_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.DeleteBook(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/doesuserstillborrow/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.DoesUserStillBorrow(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        if !res.ResponseBool {
            return c.JSON(fiber.Map{
                "response_bool": false,
            })
        } else {
            return c.JSON(res)
        }
    })

    app.Post("/createborrow", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        bookIDInt, err := strconv.Atoi(c.FormValue("book_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert book_id to int")
        }
        userIDInt, err := strconv.Atoi(c.FormValue("user_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert user_id to int")
        }
        req := &proto.Borrow{
        	BookId:       int32(bookIDInt),
        	UserId:       int32(userIDInt),
        	BorrowedDate: c.FormValue("borrowed_date"),
        	ReturnDate:   c.FormValue("return_date"),
        	ReturnedDate: c.FormValue("returned_date"),
        	Returned:     c.FormValue("returned"),
        }

        res, err := bookClient.CreateBorrow(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/createreturn", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        borrowIDInt, err := strconv.Atoi(c.FormValue("borrow_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert borrow_id to int")
        }

        req := &proto.IntRequest{
        	RequestInt: int32(borrowIDInt),
        }

        res, err := bookClient.CreateReturn(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getborrowings", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        minInt, err := strconv.Atoi(c.FormValue("min"))
        if err != nil {
            return c.Status(500).SendString("failed to convert min to int")
        }
        maxInt, err := strconv.Atoi(c.FormValue("max"))
        if err != nil {
            return c.Status(500).SendString("failed to convert max to int")
        }
        req := &proto.IDLimits{
        	Min: int32(minInt),
        	Max: int32(maxInt),
        }

        res, err := bookClient.GetBorrowings(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getborrowingsbydate", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        // validate start and end dates
        _, err := time.Parse("2006-01-02", c.FormValue("start_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse start_date")
        }
        _, err = time.Parse("2006-01-02", c.FormValue("end_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse end_date")
        }
        req := &proto.DateLimits{
        	StartDate: c.FormValue("start_date"),
        	EndDate:   c.FormValue("end_date"),
        }

        res, err := bookClient.GetBorrowingsByDate(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getborrowingsbyuserid/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.GetBorrowingsByUserID(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getreturns", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        minInt, err := strconv.Atoi(c.FormValue("min"))
        if err != nil {
            return c.Status(500).SendString("failed to convert min to int")
        }
        maxInt, err := strconv.Atoi(c.FormValue("max"))
        if err != nil {
            return c.Status(500).SendString("failed to convert max to int")
        }
        req := &proto.IDLimits{
        	Min: int32(minInt),
        	Max: int32(maxInt),
        }

        res, err := bookClient.GetReturns(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getreturnsbydate", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        // validate start and end dates
        _, err := time.Parse("2006-01-02", c.FormValue("start_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse start_date")
        }
        _, err = time.Parse("2006-01-02", c.FormValue("end_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse end_date")
        }

        req := &proto.DateLimits{
        	StartDate: c.FormValue("start_date"),
        	EndDate:   c.FormValue("end_date"),
        }

        res, err := bookClient.GetReturnsByDate(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Get("/getreturnsbyuserid/:id", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        idInt, err := strconv.Atoi(c.Params("id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(idInt),
        }

        res, err := bookClient.GetReturnsByUserID(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getoverdues", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        // validate start and end dates
        _, err := time.Parse("2006-01-02", c.FormValue("start_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse start_date")
        }
        _, err = time.Parse("2006-01-02", c.FormValue("end_date"))
        if err != nil {
            return c.Status(500).SendString("failed to parse end_date")
        }
        req := &proto.DateLimits{
        	StartDate: c.FormValue("start_date"),
        	EndDate:   c.FormValue("end_date"),
        }

        res, err := bookClient.GetOverdues(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/editborrow", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        borrowingIDInt, err := strconv.Atoi(c.FormValue("borrowing_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert borrowing_id to int")
        }
        newBookIDInt, err := strconv.Atoi(c.FormValue("new_book_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert new_book_id to int")
        }
        newUserIDInt, err := strconv.Atoi(c.FormValue("new_user_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert new_user_id to int")
        }
        req := &proto.UpdateBorrow{
        	BorrowingId:     int32(borrowingIDInt),
        	NewBookId:       int32(newBookIDInt),
        	NewUserId:       int32(newUserIDInt),
        	NewBorrowedDate: c.FormValue("new_borrowed_date"),
        	NewReturnDate:   c.FormValue("new_return_date"),
        	NewReturnedDate: c.FormValue("new_returned_date"),
        	NewReturned:     c.FormValue("new_returned"),
        }

        res, err := bookClient.EditBorrow(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/deleteborrow", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        borrowingIDInt, err := strconv.Atoi(c.FormValue("borrowing_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert borrowing_id to int")
        }
        req := &proto.IntRequest{
        	RequestInt: int32(borrowingIDInt),
        }

        res, err := bookClient.DeleteBorrow(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })

    app.Post("/getbookrecommendations", func(c *fiber.Ctx) error {
        bearerToken := c.Get("Authorization")
        ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", bearerToken)
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        // INPUT
        userIDInt, err := strconv.Atoi(c.FormValue("category_id"))
        if err != nil {
            return c.Status(500).SendString("failed to convert category_id to int")
        }
        limitInt, err := strconv.Atoi(c.FormValue("limit"))
        if err != nil {
            return c.Status(500).SendString("failed to convert limit to int")
        }
        req := &proto.GetRecommendation{
        	CategoryId: int32(userIDInt),
        	Limit:      int32(limitInt),
        }

        res, err := bookClient.GetBookRecommendations(ctx, req)
        if err != nil {
            return c.Status(500).SendString("Error calling BookService: " + err.Error())
        }

        return c.JSON(res)
    })


    // fiber rest
    log.Fatal(app.Listen("0.0.0.0:3000"))
}
