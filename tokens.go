package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/twinj/uuid"
)

//TokenDetails - details to tokens
type TokenDetails struct {
	AccessToken  string `json:"access_token" bson:"access_token"`
	RefreshToken string `json:"refresh_token" bson:"refresh_token"`
	AccessUUID   string `json:"-" bson:"-"`
	RefreshUUID  string `json:"-" bson:"-"`
	AtExpires    int64  `json:"-" bson:"-"`
	RtExpires    int64  `json:"-" bson:"-"`
}

//AccessDetails -
type AccessDetails struct {
	AccessUUID string
	UID        string
}

/*CreateTokens - creates
*
*
*
 */
func CreateTokens(uid string) (*TokenDetails, error) {
	//TODO - increased time from 15min / 24hours to 60min /30days to ease testing
	var err error
	td := &TokenDetails{}
	// Access Token
	td.AtExpires = time.Now().Add(time.Minute * 60).Unix()
	td.AccessUUID = uuid.NewV4().String()
	// Refresh Token
	td.RtExpires = time.Now().Add(time.Hour * 24 * 30).Unix()
	td.RefreshUUID = uuid.NewV4().String()

	// Creating Access Token
	os.Setenv("ACCESS_SECRET", "SECRET") // This should be in an env file!!!!!
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUUID
	atClaims["uid"] = uid
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}

	// Creating Refresh Token
	os.Setenv("REFRESH_SECRET", "SECRET") // This should be in an env file!!!!!
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUUID
	rtClaims["uid"] = uid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, err
	}

	// Store both UUIDs in Redis
	err = StoreTokens(uid, td)
	if err != nil {
		return nil, err
	}

	// Success
	return td, nil
}

/*StoreTokens - store
*
*
*
 */
func StoreTokens(uid string, td *TokenDetails) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Converting Unix to UTC(to Time object)
	at := time.Unix(td.AtExpires, 0)
	rt := time.Unix(td.RtExpires, 0)
	now := time.Now()

	// Insert AccessUUID into Redis
	errAccess := tokensClient.Set(ctx, td.AccessUUID, td.RefreshUUID, at.Sub(now)).Err()
	if errAccess != nil {
		return errAccess
	}
	// Insert RefreshUUID into Redis
	errRefresh := tokensClient.Set(ctx, td.RefreshUUID, uid, rt.Sub(now)).Err()
	if errRefresh != nil {
		return errRefresh
	}
	// Success
	return nil
}

/*ExtractTokenMetadata - ex
*
*
*
 */
func ExtractTokenMetadata(r *http.Request) (*AccessDetails, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse access token
	token, err := ParseToken(r, "ACCESS_TOKEN")

	// Verify token parsing error
	if err != nil || token == nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			// Check if the error is ValidationErrorExpired
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New("Access token expired")
			}
		}
		return nil, errors.New("Invalid token")
	}

	// If valid, get the payload:
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// The token claims should conform to MapClaims
		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, errors.New("Invalid token")
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			return nil, errors.New("Invalid token")
		}
		_, err := tokensClient.Get(ctx, accessUUID).Result()
		if err != nil {
			return nil, errors.New("Access token expired")
		}
		// Success
		return &AccessDetails{
			AccessUUID: accessUUID,
			UID:        uid,
		}, nil
	}
	return nil, errors.New("Invalid token")
}

/*ParseToken - parse
*
*
*
 */
func ParseToken(r *http.Request, tokenType string) (*jwt.Token, error) {
	// Extract token string
	tokenString := ExtractToken(r)
	// Verify the token
	os.Setenv(tokenType, "SECRET") // This should be in an env file!!!!!
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv(tokenType)), nil
	})
	return token, err
}

/*ExtractToken - does
*
*
*
 */
func ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		// Success
		return strArr[1]
	}
	return ""
}

/*DeleteAuth - delete
*
*
*
 */
func DeleteAuth(givenUUID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deleted, err := tokensClient.Del(ctx, givenUUID).Result()
	if err != nil {
		return 0, err
	}
	// Success
	return deleted, nil
}

/*Logout - logout
*
*
*
 */
func Logout(accessUUID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	refreshUUID, err := tokensClient.Get(ctx, accessUUID).Result()
	if err != nil {
		return 0, err
	}
	deleted, err := DeleteAuth(accessUUID)
	if err != nil {
		return 0, err
	}
	deleted, err = DeleteAuth(refreshUUID)
	if err != nil {
		return 0, err
	}
	// Success
	return deleted, nil
}

/*RefreshToken - ref
*
*
*
 */
func RefreshToken(r *http.Request) Response {
	// Parse refresh token
	token, err := ParseToken(r, "REFRESS_TOKEN")

	// Verify token parsing error
	if err != nil || token == nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			// Check if the error is ValidationErrorExpired
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return Response{Error: true, Msg: "Refresh token expired"}
			}
		}
		return Response{Error: true, Msg: "Invalid token"}
	}

	// If valid, get the payload:
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// The token claims should conform to MapClaims
		refreshUUID, ok := claims["refresh_uuid"].(string)
		if !ok {
			return Response{Error: true, Msg: "Invalid token"}
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			return Response{Error: true, Msg: "Invalid token"}
		}
		// Delete the previous Refresh Token
		deleted, delErr := DeleteAuth(refreshUUID)
		if delErr != nil || deleted == 0 {
			return Response{Error: true, Msg: "Refresh token expired"}
		}
		// Create new pairs of refresh and access tokens
		td, createErr := CreateTokens(uid)
		if createErr != nil {
			return Response{Error: true, Msg: "An error occurred"}
		}

		// Success
		tokensJSON, err := json.Marshal(td)
		if err != nil {
			return Response{Error: true, Msg: "error processing information"}
		}
		return Response{Error: false, Data: tokensJSON}

	}
	return Response{Error: true, Msg: "Invalid token"}
}
