package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

func userRemoveFriend(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	otherUID := mux.Vars(req)["UID"]
	err = DBRemoveFriend(selfUID, otherUID)
	return Response{Error: false, Msg: "success"}

}

func userListFriend(req *http.Request) Response {

	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	res, err := DBListFriend(selfUID)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	dataRes, err := json.Marshal(bson.M{"friend_list": res}) // res is an array with all the friends
	if err != nil {
		return Response{Error: true, Msg: err.Error(), Data: dataRes}
	}

	return Response{Error: false, Msg: "success", Data: dataRes}

}

func userSendRequest(req *http.Request) Response {

	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	otherUID := mux.Vars(req)["UID"]
	err = DBSendRequest(selfUID, otherUID)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	return Response{Error: false, Msg: "success"}
}

func userAcceptRequest(req *http.Request) Response {

	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	otherUID := mux.Vars(req)["UID"]
	err = DBAcceptRequest(otherUID, selfUID) // sender, receiver
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	return Response{Error: false, Msg: "success"}

}

func userRefuseRequest(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	otherUID := mux.Vars(req)["UID"]

	err = DBRefuseRequest(otherUID, selfUID)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	return Response{Error: false, Msg: "success"}

}

func userListRequest(req *http.Request) Response {
	tokenAuth, err := ExtractTokenMetadata(req)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	selfUID := tokenAuth.UID
	res, err := DBListRequest(selfUID)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}
	dataRes, err := json.Marshal(res)
	if err != nil {
		return Response{Error: true, Msg: err.Error()}
	}

	return Response{Error: false, Msg: "success", Data: dataRes}

}
