package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//Request - request form made to the API
type Request struct {
	Token string `json:"token" bson:"token"`
	Data  string `json:"data" bson:"data"`
}

//Response - response given by the server to the request -- TODO: se calhhar adicionar variavel tipo "error" ? ou é mau passar o erro verdadeiro?
type Response struct {
	Error bool            `json:"error" bson:"error"`
	Msg   string          `json:"msg,omitempty" bson:"msg"`
	Data  json.RawMessage `json:"data,omitempty" bson:"data"`
}

//TODO: ORGANIZE THIS ! MAYBE FILE FOR STRUCTS AND GLOBALS? main??

/*root - does nothing
*
 */
func root(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Invalid request")
}

/*userLogin - handler for login requests
*
 */
func userLoginEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userLogin(req)
	json.NewEncoder(w).Encode(res)
}

/*userLogout - handler for logout requests
*
 */
func userLogoutEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userLogout(req)
	json.NewEncoder(w).Encode(res)

}

/*userPing - handler for ping requests
*
 */
func userPingEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userPing(req)
	json.NewEncoder(w).Encode(res)
}

/*userRefreshToken - handler for refreshtoken requests
* Responses are handled in tokens.go for this function
 */
func userRefreshTokenEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := RefreshToken(req)
	json.NewEncoder(w).Encode(res)
	return
}

/*userSignup - handler for signup requests
*
 */
func userSignupEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userSignup(req)
	json.NewEncoder(w).Encode(res)

}

/*
*
 */
func userLocationEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userLocation(req)
	json.NewEncoder(w).Encode(res)

}

/*createMsg - handler for creating message requests
* It receives the text of the message, the UID of the writer, the date the latitude and longitude.
 */
func createMsgEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := createMsg(req)
	json.NewEncoder(w).Encode(res)

}

/*reqMsgZone - handler for requests to all the messages in a zone
*
 */
func reqMsgZoneEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := reqMsgZone(req)
	json.NewEncoder(w).Encode(res)

}

/*UpdateLikes - handler to udpate the number of likes/dislikes in a message
*
 */
func updateEvalEP(w http.ResponseWriter, req *http.Request) { //incomplete
	w.Header().Set("Content-Type", "application/json")
	res := updateEval(req)
	json.NewEncoder(w).Encode(res)
}

/*
*
 */
func getEvalEP(w http.ResponseWriter, req *http.Request) { //incomplete
	w.Header().Set("Content-Type", "application/json")
	res := getEval(req)
	json.NewEncoder(w).Encode(res)
}

func userRemoveFriendEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userRemoveFriend(req)
	json.NewEncoder(w).Encode(res)
}

func userListFriendEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userListFriend(req)
	json.NewEncoder(w).Encode(res)
}

func userSendRequestEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userSendRequest(req)
	json.NewEncoder(w).Encode(res)
}
func userAcceptRequestEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userAcceptRequest(req)
	json.NewEncoder(w).Encode(res)
}

func userRefuseRequestEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userRefuseRequest(req)
	json.NewEncoder(w).Encode(res)
}

func userListRequestEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userListRequest(req)
	json.NewEncoder(w).Encode(res)
}

func userImagesEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "multipart/form-data")
	res := userImages(req)
	json.NewEncoder(w).Encode(res)
}

func userUpdateUsernameEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userUpdateUsername(req)
	json.NewEncoder(w).Encode(res)
}

func userValidateEP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := userValidate(req)
	json.NewEncoder(w).Encode(res)
}

/*main - main is main
*
 */
func main() {

	// logging levels: Trace, Debug, Info, Warning, Error, Fatal and Panic.
	// Logs
	file, err := os.OpenFile("logs/info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)
	log.SetFormatter(&log.JSONFormatter{}) // &log.TextFormatter
	log.SetLevel(log.InfoLevel)

	// TODO - change names to: users, logins logouts pings refreshes signups lists removes accepts refuses sends ? also change createMsg reqMsgZone?
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", root)
	//change latitude and longitude to query parameters
	router.HandleFunc("/messages/near", reqMsgZoneEP).Queries("latitude", "", "longitude", "", "order", "{order:new|best}", "group", "{group:all|friends}").Methods("GET")
	router.HandleFunc("/messages/post", createMsgEP).Methods("POST")
	// TODO , change eval to query parameters. Also change any headers used to query
	router.HandleFunc("/messages/{MID}", updateEvalEP).Queries("eval", "{eval:upvote|downvote}").Methods("POST") //this one posts a like // eval can be upvote or downvote
	//not being used
	//router.HandleFunc("/messages/{MID}/{eval}", getEvalEP).Methods("GET")     // this one gets the likes
	router.HandleFunc("/users/login", userLoginEP).Methods("POST")
	router.HandleFunc("/users/logout", userLogoutEP).Methods("GET")
	router.HandleFunc("/users/ping", userPingEP).Methods("GET")
	router.HandleFunc("/users/signup", userSignupEP).Methods("POST")
	router.HandleFunc("/users/validate", userValidateEP).Queries("code", "").Methods("GET")
	router.HandleFunc("/users/location", userLocationEP).Methods("POST")
	router.HandleFunc("/users/token/refresh", userRefreshTokenEP).Methods("GET")
	router.HandleFunc("/users/friends/remove/{UID}", userRemoveFriendEP).Methods("POST")
	router.HandleFunc("/users/friends/list", userListFriendEP).Methods("GET")
	router.HandleFunc("/users/friends/request/send/{UID}", userSendRequestEP).Methods("POST")
	router.HandleFunc("/users/friends/request/accept/{UID}", userAcceptRequestEP).Methods("POST")
	router.HandleFunc("/users/friends/request/refuse/{UID}", userRefuseRequestEP).Methods("POST")
	router.HandleFunc("/users/friends/request/list", userListRequestEP).Methods("GET")
	router.HandleFunc("/users/images/post", userImagesEP).Methods("POST")
	fmt.Println("Server running on port 8080")

	log.Fatal(http.ListenAndServe(":8080", router))
}
