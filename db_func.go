package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

//Client -- client go mongodb
var (
	Client          *mongo.Client     = DBConnect() //probably return this and redis directly from function and assign here
	appDB           *mongo.Database   = Client.Database("message_poster_app")
	usersColl       *mongo.Collection = appDB.Collection("users")
	messagesColl    *mongo.Collection = appDB.Collection("messages")
	likesColl       *mongo.Collection = appDB.Collection("likes")
	friendshipsColl *mongo.Collection = appDB.Collection("friendships")
	friendsReqsColl *mongo.Collection = appDB.Collection("friends_requests")
)

//Collection is a handle to a MongoDB collection. It is safe for concurrent use by multiple goroutines. (from godocs -mongodb)

/*DBConnect - Returns client after making a connection
*
*
*
 */
func DBConnect() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientLocal, err := mongo.Connect(ctx, options.Client().ApplyURI(
		"mongoURI"))
	if err != nil {
		fmt.Println("error connection to DB")
		log.Fatal(err)
	}
	return clientLocal
}

/*DBGetHash - Queries DB for hash of the password to be compared in login
*	- also returns uid and if the user is validated or not
*
*
 */
func DBGetHash(email string) (string, string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result User
	projection := bson.M{"_id": 0, "uid": 1, "password": 1, "validated_account": 1} // which fields are returned?
	err := usersColl.FindOne(ctx, bson.M{"email": email}, options.FindOne().SetProjection(projection)).Decode(&result)
	fmt.Println(result) //TODO: to not receive _id it is necessary to change something in the project in mongodb
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			fmt.Println("No such user")
			return "", "", false, err
		}
		//log.Fatal(err)
		return "", "", false, err
	}

	return result.UID, result.Password, result.ValidatedAccount, nil
}

//DBValidateUser - ...
func DBValidateUser(UID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := usersColl.UpdateOne(ctx, bson.M{"uid": UID}, bson.M{"$set": bson.M{"validated_account": true}})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

/*DBExistsUser - check if a specific user exists in the DB by email
*
*
*
 */
func DBExistsUser(email string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := usersColl.CountDocuments(ctx, bson.D{primitive.E{Key: "email", Value: email}})
	if err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		return true
	}
	return false
}

/*DBInsertUser - inserts new user that signed up
*
*
*
 */
func DBInsertUser(u *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := usersColl.InsertOne(ctx, u)
	if err != nil {
		fmt.Println("Failed to insert user in DB")
		return err
		//log.Fatal(err)
	}
	id := res.InsertedID
	fmt.Println(id)
	return nil

}

/*DBCreateMessage - create the message in the DB.
*
*
 */
func DBCreateMessage(msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := messagesColl.InsertOne(ctx, msg) // maybe do not return res? options?
	if err != nil {
		return err
		//log.Fatal(err)
	}
	id := res.InsertedID

	fmt.Println(id)

	return nil

}

/*DBUpdateLocation - xxx
*
 */
func DBUpdateLocation(l *Location, uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	singleResult := usersColl.FindOneAndUpdate(ctx, bson.M{"uid": uid}, bson.M{"location": l})
	if err := singleResult.Err(); err != nil {
		//log.Fatal(err)
		return err
	}
	return nil
}

/*DBQueryMessages - Retrieve the messages in a certain radius of this latitude and longitude
*
*
 */
func DBQueryMessages(location Location, radius int, UID string, order string, group string) ([]Message, error) {
	//WHen do we update location? maybe don't need to save user location, just retrieve it when asking for the messages
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ordSet string
	if order == "new" {
		ordSet = "date"
	} else {
		ordSet = "eval_value"
	}

	// message filter check if uid of message creator is friend of uid that sent the query?
	// use aggregation to return list of UID friends? TODO

	//TODO: maybe only return the message and timestamp here?
	filter := bson.M{"location": bson.M{"$near": bson.M{"$geometry": location, "$maxDistance": radius}}}
	opts := options.Find().SetProjection(bson.M{"_id": 0, "location": 0}).SetSort(bson.M{ordSet: -1}).SetLimit(500)
	if group == "friends" { // if group is friends change filter to also consider friend list
		userFriends, err := DBListFriend(UID)
		if err != nil {
			fmt.Printf("error listing friends")
			return nil, err
		}
		filter = bson.M{"location": bson.M{"$near": bson.M{"$geometry": location, "$maxDistance": radius}}, "uid": bson.M{"$in": userFriends}}

	}

	cursor, err := messagesColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var res []Message
	if err = cursor.All(ctx, &res); err != nil {
		return nil, err
	}
	for _, d := range res {
		fmt.Println(d)
	}
	//fmt.Println(results)

	return res, nil
}

/*DBUpdateEval - changes the number of likes/dislikes in a message
*
* TODO - Is it necessary to find if exists and then insert if it does not or remove if it does?
*
 */
func DBUpdateEval(MID string, UID string, eval string) error { // eval is1 or 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var incV int
	switch eval {
	case "upvote":
		incV = 1

	case "downvote":
		incV = -1
	}
	//check if message exists
	var res bson.M
	opts := options.FindOne().SetProjection(bson.M{"_id": 0, "eval": 1})             //receive only eval
	err := likesColl.FindOne(ctx, bson.M{"mid": MID, "uid": UID}, opts).Decode(&res) // if res[eval] == 1 - like || res[eval] == 0 - dislike
	if err == mongo.ErrNoDocuments {                                                 //NO DOCUMENTS, INSERT EVALUATION
		_, err = likesColl.InsertOne(ctx, bson.M{"mid": MID, "uid": UID, "eval": eval})
		if err != nil {
			fmt.Println("db error inserting evaluation")
			return err
		}
		_, err := messagesColl.UpdateOne(ctx, bson.M{"mid": MID}, bson.M{"$inc": bson.M{"eval_value": incV}})
		if err != nil {
			fmt.Println("db error updating likes/dislikes count")
			return err
		}
		return nil // if inserted successfully, everything is done here
	} else if err != nil { // ERROR DB, return
		fmt.Println(err.Error())
		return err
	} else { // ALREADY FOUND ON DB, IF IT IS EQUAL DO FIRST ELSE, DO THE OTHER
		//case where already evaluated with the same, so remove evaluation
		if eval == res["eval"] {
			_, err := likesColl.DeleteOne(ctx, bson.M{"mid": MID, "uid": UID})
			if err != nil {
				fmt.Println("Error deleting eval")
				return err
			}
			_, err = messagesColl.UpdateOne(ctx, bson.M{"mid": MID}, bson.M{"$inc": bson.M{"eval_value": -incV}}) //do the opposite, if it has an upvote remove it
			if err != nil {
				fmt.Println("db error updating likes/dislikes count")
				return err
			}
			return nil
		}
		//case where evaluated with different value, so update
		_, err = likesColl.UpdateOne(ctx, bson.M{"mid": MID, "uid": UID}, bson.M{"$set": bson.M{"eval": eval}})
		if err != nil {
			fmt.Println("db error updating evaluation")
			return err
		}
		_, err = messagesColl.UpdateOne(ctx, bson.M{"mid": MID}, bson.M{"$inc": bson.M{"eval_value": 2 * incV}}) //do the opposite, if it has an upvote remove it
		if err != nil {
			fmt.Println("db error updating likes/dislikes count")
			return err
		}
		//update messages coll with upvote - downvote count
		return nil
	}

}

//DBGetMsgEval - NOT USED
func DBGetMsgEval(MID string, eval int) ([]primitive.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := likesColl.Find(ctx, bson.M{"mid": MID, "eval": eval})
	if err != nil {
		return nil, err
	}

	var res []bson.M
	if err = cursor.All(ctx, &res); err != nil {
		return nil, err
	}
	return res, nil
}

/*DBGetUser - Returns user from UID, check privacy
*
* Need to check if the profile being asked for is from the person himself or is from someone else, this changes which fields should be returned
* incomplete
 */
func DBGetUser(UID string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u User
	err := usersColl.FindOne(ctx, bson.M{"uid": UID}).Decode(&u)
	if err != nil {
		return nil, err
		//log.Fatal(err) //should not be fatal
	}

	return &u, nil
}

/*
* From here on out DB friends functions
*
*
 */

//DBRemoveFriend - ...
func DBRemoveFriend(UID1, UID2 string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// if user1 and user2 are friends
	_, err := friendshipsColl.DeleteOne(ctx, bson.M{"$or": []bson.M{bson.M{"uid1": UID1, "uid2": UID2}, bson.M{"uid1": UID2, "uid2": UID1}}}) //see if correct wit hkeys
	if err != nil {
		fmt.Println("Failed to remove friendship in DB")
		return err
		//log.Fatal(err)
	}
	return nil

}

//DBListFriend - ...
func DBListFriend(UID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := friendshipsColl.Find(ctx, bson.M{"$or": []bson.M{bson.M{"uid1": UID}, bson.M{"uid2": UID}}}) // if user is one of the elements of friendship
	if err != nil {
		fmt.Println("Failed to find any ") // should return empty, not error!
		return nil, err
		//log.Fatal(err)
	}
	var res []bson.M
	if err = cursor.All(ctx, &res); err != nil {
		return nil, err
	}
	var resString []string
	for _, el := range res {
		if el["uid1"] == UID {
			resString = append(resString, el["uid2"].(string))
		} else {
			resString = append(resString, el["uid1"].(string))
		}
	}
	fmt.Println(res)
	fmt.Println(resString)
	return resString, nil

}

//DBSendRequest - ....
func DBSendRequest(senderUID, receiverUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if val, err := checkUserExists(receiverUID); !val {
		if err != nil {
			return err
		}
		return errors.New("UID does not exist")
	}
	singleRes := friendshipsColl.FindOne(ctx, bson.M{"$or": []bson.M{bson.M{"uid1": senderUID, "uid2": receiverUID}, bson.M{"uid1": receiverUID, "uid2": senderUID}}})
	if err := singleRes.Err(); err != mongo.ErrNoDocuments { // if error is no documents, just move on, friendship does not exist
		if err != nil { // if there is no error, friend already exists so quit here
			return err // if standard error, just send it. If error is ErrNoDocuments, continue
		}
		return errors.New("Request sent to user that is already your friend")

	}

	//check if this person already sent u a request
	singleRes = friendsReqsColl.FindOne(ctx, bson.M{"sender_uid": receiverUID, "receiver_uid": senderUID})
	if err := singleRes.Err(); err != mongo.ErrNoDocuments { // if err is no documents, person did not send a request, so we send it
		if err != nil { // if there is no error, person sent u a request, so accept it
			return err // if it is a random error, return it

		}
		//if there is a request in the opposite direction, accept their request
		err := DBAcceptRequest(receiverUID, senderUID)
		if err != nil {
			return err
		}

	}

	// if there is no request on the opposite direction, send one!
	//this already verifies if it exists, since the key is unique!
	_, err := friendsReqsColl.InsertOne(ctx, bson.M{"sender_uid": senderUID, "receiver_uid": receiverUID})
	if err != nil {
		fmt.Println("failed to send request")
		return err
	}
	return nil

}

//DBAcceptRequest - ... // this actually corresponds to AddFriend, since u are accepting requests
func DBAcceptRequest(senderUID, receiverUID string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//delete one or find one and delete? depends on key?
	deleteRes, err := friendsReqsColl.DeleteOne(ctx, bson.M{"receiver_uid": receiverUID})
	if err != nil {
		fmt.Println("Failed to delete request")
		return err
		//log.Fatal(err)
	}
	if deleteRes.DeletedCount == 0 {
		fmt.Println("request does not exist, can't accept")
		return errors.New("request does not exist, can't accept")
	}

	_, err = friendshipsColl.InsertOne(ctx, bson.M{"uid1": senderUID, "uid2": receiverUID}) //can change to camel case later, change other structs bson:
	if err != nil {
		fmt.Println("Failed to insert new friendship")
		return err
		//log.Fatal(err)
	}
	return nil

}

//DBRefuseRequest - ....
func DBRefuseRequest(senderUID, receiverUID string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deleteRes, err := friendsReqsColl.DeleteOne(ctx, bson.M{"sender_uid": senderUID, "receiver_uid": receiverUID}) //can change to camel case later, change other structs bson:
	if err != nil {
		fmt.Println("Failed to delete request")
		return err
	}
	if deleteRes.DeletedCount != 1 {
		return errors.New("Cannot refuse unexistent request")
	}
	return nil

}

//DBListRequest - ....
func DBListRequest(UID string) ([]primitive.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := friendsReqsColl.Find(ctx, bson.M{"receiver_uid": UID})
	if err != nil {
		fmt.Println("Failed to find any friend requests ") // should return empty, not error!
		return nil, err
	}

	var res []bson.M
	if err = cursor.All(ctx, &res); err != nil {
		return nil, err
	}
	return res, nil

}

//DBCheckUserEval - check if user evaluated this message and how he did it
func DBCheckUserEval(UID string, MID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var res bson.M
	projection := bson.M{"_id": 0, "eval": 1} // which fields are returned?
	err := likesColl.FindOne(ctx, bson.M{"mid": MID, "uid": UID}, options.FindOne().SetProjection(projection)).Decode(&res)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "empty", nil // no likes/dislikes
		}
		return "", err
	}

	return res["eval"].(string), nil
}

//DBUpdateUsername - updates username
func DBUpdateUsername(UID string, newUsername string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var res bson.M
	proj := bson.M{"_id": 0, "last_changed_name": 1}
	err := usersColl.FindOne(ctx, bson.M{"uid": UID}, options.FindOne().SetProjection(proj)).Decode(&res)
	if err != nil {
		fmt.Println(err)
		return err
	}
	currTime := time.Now().Unix()
	if currTime-res["last_changed_name"].(int64) < (3600 * 24 * 7) {
		return errors.New("Can only change username once every 7 days")
	}

	_, err = usersColl.UpdateOne(ctx, bson.M{"uid": UID}, bson.M{"username": newUsername, "last_changed_name": currTime})
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

/*
* AUX FUNCTIONS
 */

func checkUserExists(UID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	singleResult := usersColl.FindOne(ctx, bson.M{"uid": UID})
	if err := singleResult.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		fmt.Println("could not req find checkuserexists ") // should return empty, not error!
		return false, err
	}
	return true, nil
}

//unused, just to create index
func createIndex() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	indexOpts := options.CreateIndexes().
		SetMaxTime(time.Second * 10)
	// Index to location 2dsphere type.
	pointIndexModel := mongo.IndexModel{
		Options: options.Index().SetBackground(true),
		Keys:    bsonx.MDoc{"location": bsonx.String("2dsphere")},
	}
	pointIndexes := messagesColl.Indexes()
	_, err := pointIndexes.CreateOne(
		ctx,
		pointIndexModel,
		indexOpts,
	)
	if err != nil {
		return err
	}
	return nil
}
