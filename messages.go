package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

//Message - a message
type Message struct {
	MID       string   `json:"mid" bson:"mid"`
	UID       string   `json:"uid" bson:"uid"`
	Title     string   `json:"title,omitempty" bson:"title" validate:"required,min=1,max=50"`
	Text      string   `json:"text,omitempty" bson:"text" validate:"required,min=1,max=500"`
	Image     string   `json:"image,omitempty" bson:"image,omitempty" validate:"omitempty,base64"`
	Date      int64    `json:"date,omitempty" bson:"date"`
	Location  Location `json:"-" bson:"location"`
	Latitude  float64  `json:"latitude" bson:"latitude"`
	Longitude float64  `json:"longitude" bson:"longitude"`
	EvalValue int      `json:"eval_value" bson:"eval_value"`
	UserEval  string   `json:"user_eval,omitempty" bson:"-"`
}

//Location - Type is normally "Point"
type Location struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"` // first longitude then latitude
}

/*
*
 */
func createMsg(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}

	}

	//decode message
	decoder := json.NewDecoder(req.Body)
	var msg Message
	err = decoder.Decode(&msg)
	if err != nil {
		fmt.Println("Failed to read request.")
		return Response{Error: true, Msg: "Failed to read request."}

	}

	if !_validateInput(msg) {
		fmt.Println("Message sent was invalid. Post Failed")
		return Response{Error: true, Msg: "Message sent was invalid. Post Failed"}

	}

	//add missing camps to the message
	msg.Date = time.Now().Unix()
	msg.MID = "m" + ksuid.New().String()
	msg.UID = tokenAuth.UID
	msg.Location = Location{Type: "Point", Coordinates: []float64{msg.Longitude, msg.Latitude}}
	msg.EvalValue = 0

	if msg.Image != "" {
		imageURL, err := UploadBytesToBlob(JpegToBytes(b64ToJpeg(msg.Image)))
		if err != nil {
			fmt.Println("error uploading image:", err)
		}
		msg.Image = imageURL
		fmt.Println(imageURL)
	}

	fmt.Println("message posted")
	if DBCreateMessage(&msg) != nil {
		fmt.Println(err.Error())
		return Response{Error: true, Msg: "Error in the DB"}
	}

	return Response{Error: false, Msg: "Message posted successfully"}

}

/*reqMsgZone-
* Request messages to DB from current zone and all adjacent zones (including diagonals)
* Considering intervals of
 */
func reqMsgZone(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}

	}
	DBListFriend(tokenAuth.UID)

	qParams := req.URL.Query()
	latstr := qParams.Get("latitude")
	longstr := qParams.Get("longitude")
	order := qParams.Get("order")
	group := qParams.Get("group")

	//latstr := req.Header.Get("latitude")
	//longstr := req.Header.Get("longitude")

	latitude, err := strconv.ParseFloat(latstr, 8)
	if err != nil {
		fmt.Println(err)
		return Response{Error: true, Msg: "Latitude Invalid"}

	}
	longitude, err := strconv.ParseFloat(longstr, 8)
	if err != nil {
		fmt.Println(err)
		return Response{Error: true, Msg: "Longitude Invalid"}

	}
	if latitude < -90.0 || latitude > 90.0 || longitude < -180.0 || longitude > 180.0 {
		fmt.Println("Coordinates are invalid. Please return to using earth coordinates.")
		return Response{Error: true, Msg: "Coordinates are invalid. Please return to using earth coordinates."}

	}
	UID := tokenAuth.UID

	location := Location{Type: "Point", Coordinates: []float64{longitude, latitude}}

	results, err := DBQueryMessages(location, 1000000, UID, order, group)
	if err != nil {
		log.WithFields(log.Fields{
			"uid": UID, "request": "reqMsgZone",
		}).Info(err)
		fmt.Println(err.Error())
		return Response{Error: true, Msg: "Error in the database"}
	}
	// add what they eval'd in that msg
	for i := range results {
		r := &results[i]
		eval, err := DBCheckUserEval(UID, r.MID)
		if err != nil {
			return Response{Error: true, Msg: "Error in the database"}
		}
		r.UserEval = eval
	}

	dataResp, err := json.Marshal(results)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	return Response{Error: false, Msg: "Request successfully completed", Data: dataResp}
}

/*
*
 */
func updateEval(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}

	}
	eval := req.URL.Query().Get("eval")
	MID := mux.Vars(req)["MID"]
	//MID := params["MID"]
	//eval := params["eval"] //This can be either 0 or 1

	UID := tokenAuth.UID
	fmt.Println(MID, eval)

	err = DBUpdateEval(MID, UID, eval)
	if err != nil {
		return Response{Error: true, Msg: "Could not Like/Dislike this message"}
	}
	return Response{Error: false, Msg: "Likes/Dislikes updated successfully"}
}

/*
*
 */
func getEval(req *http.Request) Response {
	MID := mux.Vars(req)["MID"]
	eval, err := strconv.Atoi(mux.Vars(req)["eval"]) //This can be either 0 or 1
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	res, err := DBGetMsgEval(MID, eval)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	dataRes, err := json.Marshal(res)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	return Response{Error: false, Data: dataRes, Msg: "Retrieved eval list successfully"}
}
