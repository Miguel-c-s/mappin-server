package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mailgun/mailgun-go"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"github.com/twinj/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/go-playground/validator.v9"
)

//DOMAIN - email domain
const DOMAIN = "hadrons.xyz"

//APIKEY - mailgun api key
const APIKEY = "key"

//User - a user
type User struct {
	UID              string `json:"uid,omitempty" bson:"uid"`
	Username         string `json:"username" bson:"username" validate:"required,min=2,max=10"`
	Email            string `json:"email" bson:"email" validate:"required,email"`
	Image            string `json:"image,omitempty" bson:"image"`
	Password         string `json:"password" bson:"password" validate:"required,min=8"` // need to add verification for password strength
	CreatedAt        int64  `json:"created_at,omitempty" bson:"created_at"`
	LastAccess       int64  `json:"last_access,omitempty" bson:"last_access"`
	LastChangedName  int64  `json:"last_changed_name,omitempty" bson:"last_changed_name"`
	ValidatedAccount bool   `json:"validated_account" bson:"validated_account"`
	//email verified bool
	// username last change date
}

//Login - variables to login into the app
type Login struct {
	Email    string `json:"email" bson:"email" validate:"required,email"`
	Password string `json:"password" bson:"password" validate:"required"`
}

// Internal

/*_validateInput - auxiliar function to validate structure
*
 */
func _validateInput(s interface{}) bool {
	v := validator.New()
	fmt.Println(s)
	err := v.Struct(s)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			fmt.Println(e)
		}
		return false //did not pass validation
	}
	return true //validated
}

/*
*
 */
func userLogin(req *http.Request) Response {
	var data Login
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return Response{Error: true, Msg: "Invalid request"}
	}
	if !_validateInput(data) {
		log.WithFields(log.Fields{
			"email": data.Email, "password": data.Password,
		}).Info("Unvalidated login info")
		fmt.Println("Login failed.")
		return Response{Error: true, Msg: "Invalid login data"}
	}

	// Get the password hash from DB
	UID, hashedPassword, validated, err := DBGetHash(data.Email)
	if err != nil {
		log.WithFields(log.Fields{
			"uid": UID,
		}).Info("Invalid login data")
		return Response{Error: true, Msg: "Invalid login data"}
	}
	if !validated {
		if codeToEmail(data.Email, UID) != nil {
			return Response{Error: true, Msg: "failed to send email with code"}
		}
		return Response{Error: true, Msg: "Account is not validated. Please check your email to confirm your account"}
	}

	// Compare the password and the stored hash
	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(data.Password)) == nil {
		// Create Access and Refresh tokens
		tokens, err := CreateTokens(UID)
		if err != nil {
			return Response{Error: true, Msg: "An error occurred, please try again"}
		}
		// Success
		log.WithFields(log.Fields{
			"uid": UID,
		}).Info("Successfully logged in")
		tokensJSON, err := json.Marshal(tokens)
		if err != nil {
			return Response{Error: true, Msg: "error processing information"}
		}
		return Response{Error: false, Data: tokensJSON}

	}
	return Response{Error: true, Msg: "Invalid login data"}
}

/*
*
 */
func userLogout(req *http.Request) Response {
	// Extract access token payload auth data
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	// Delete the access token UUID from Redis, to delete this session
	deleted, delErr := Logout(tokenAuth.AccessUUID)
	if delErr != nil || deleted == 0 {
		return Response{Error: true, Msg: "Unauthorized"}
	}
	// Success
	log.WithFields(log.Fields{
		"uid": tokenAuth.UID,
	}).Info("Successfully logged out")
	return Response{Error: false, Msg: "Successfully logged out"}
}

/*userPing - used to check if acccess token is valid when starting app
*
 */
func userPing(req *http.Request) Response {

	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		print("token invalid")
		return Response{Error: true, Msg: err.Error()}
	}

	uidJSON, err := json.Marshal(tokenAuth.UID)
	if err != nil {
		return Response{Error: true, Msg: "Invalid uid"}
	}
	print("success")
	return Response{Error: false, Data: uidJSON}

}

/*
*
 */
func userValidate(req *http.Request) Response {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	qParams := req.URL.Query()
	code := qParams.Get("code")
	UID, err := codesClient.Get(ctx, code).Result()
	if err != nil {
		return Response{Error: true, Msg: "code does not exist or expired"}
	}
	if DBValidateUser(UID) != nil {
		return Response{Error: true, Msg: "db error validating user"}

	}
	return Response{Error: false}

}

/*
*
 */
func userSignup(req *http.Request) Response {

	decoder := json.NewDecoder(req.Body)
	var u1 User
	err := decoder.Decode(&u1)
	if err != nil {
		panic(err)
	}
	if !_validateInput(u1) {
		log.WithFields(log.Fields{
			"email": u1.Email,
		}).Info("Invalid signup data")
		return Response{Error: true, Msg: "Information Invalid. Signup failed."}

	}

	if DBExistsUser(u1.Email) {
		log.WithFields(log.Fields{
			"email": u1.Email,
		}).Info("Email already used by another user.")
		return Response{Error: true, Msg: "Email already used by another user."}

	}
	u1.CreatedAt = time.Now().Unix()
	u1.LastAccess = time.Now().Unix()
	u1.UID = "u" + ksuid.New().String()
	u1.ValidatedAccount = false

	hash, err := bcrypt.GenerateFromPassword([]byte(u1.Password), 12)
	u1.Password = string(hash)
	if DBInsertUser(&u1) != nil {
		return Response{Error: true, Msg: "DB Error"}

	}
	//send verification e-mail
	if codeToEmail(u1.Email, u1.UID) != nil {
		return Response{Error: true, Msg: "failed to send email with code"}

	}

	return Response{Error: false, Msg: "Account created successfully"}
}

/*
*
 */
func userLocation(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	decoder := json.NewDecoder(req.Body)
	var l Location
	err = decoder.Decode(&l)
	if err != nil {
		fmt.Println("Failed to read request.")
		return Response{Error: true, Msg: "Failed to read request."}
	}
	if !_validateInput(l) {
		fmt.Println("Message sent was invalid. Post Failed")
		return Response{Error: true, Msg: "Message sent was invalid. Post Failed"}

	}
	UID := tokenAuth.UID
	DBUpdateLocation(&l, UID)
	return Response{Error: false, Msg: "location updated successfully"}
}

func userImages(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	req.ParseMultipartForm(32 << 20) //32 times 2^20, 32 MB

	file, _, err := req.FormFile("image")
	if err != nil {
		return Response{Error: true, Msg: "can't read image"}

	}
	defer file.Close()

	//send image to image server
	//return link
	// save link in user.image

	destination, err := os.Create(fmt.Sprintf("images/image_%s.jpg", tokenAuth.UID))
	if err != nil {
		return Response{Error: true, Msg: "can't save image"}
	}
	io.Copy(destination, file)

	return Response{Error: false}
}

/*
*
 */
func userUpdateUsername(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	UID := tokenAuth.UID
	var res bson.M
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&res)
	if err != nil {
		fmt.Println("Failed to read request.")
		return Response{Error: true, Msg: "Failed to read request."}
	}
	u, ok := res["new_username"].(string)
	if !ok {
		return Response{Error: true, Msg: "invalid username"}
	}
	DBUpdateUsername(UID, u)
	return Response{Error: false}
}

/*
*
 */
func sendEmail(email, code string) (string, error) {
	mg := mailgun.NewMailgun(DOMAIN, APIKEY)
	m := mg.NewMessage(
		"Mappin App <mappin@hadrons.xyz>",
		"Confirm your account",
		"Use this code: "+code,
		email,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}

/*
 */
func codeToEmail(email, uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	code := "v" + uuid.NewV4().String()
	duration, err := time.ParseDuration("24h")
	if err != nil {
		return err
	}
	codesClient.Set(ctx, code, uid, duration)
	url := "https://mappin.hadrons.xyz/users/validate?code=" + code
	_, err = sendEmail(email, url) // used this bcs independent of time. It is not good to generate tokens that are time dependent.
	if err != nil {
		return err
	}
	return nil
}
