package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

var (
	ctx    context.Context
	client *firestore.Client
)

const callsPerMin int64 = 2

func init() {
	ctx = context.Background()
	client = createClient()
}

func createClient() *firestore.Client {
	// Sets your Google Cloud Platform project ID.
	projectID := "story-time-337102"

	// [END firestore_setup_client_create]
	// Override with -project flags
	// flag.StringVar(&projectID, "project", projectID, "The Google Cloud Platform project ID.")
	// flag.Parse()

	// [START firestore_setup_client_create]
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// Close client when done with
	// defer client.Close()
	return client
}

type tooManyRequestsError struct {
	cooldown int
}

func (e *tooManyRequestsError) Error() string {
	return fmt.Sprintf("Whoa there, you exceeded your quota of %d requests per minute, please wait %d seconds. ", callsPerMin, e.cooldown)
}

func userAccess(userID string, username string) error {
	userDoc := client.Collection("users").Doc(userID)
	dsnap, err := userDoc.Get(ctx)
	totalCalls := int64(1)
	minuteCalls := int64(1)
	lastAccessed := time.Now().Unix()
	// if err != nil {
	// 	log.Println("UNABLE TO GET DOC",dsnap.Exists())
	// 	return err
	// }
	if dsnap == nil {
		log.Println("data snap is null")
		return err
	}

	if dsnap.Exists() {
		dmap := dsnap.Data()
		totalCalls, _ = dmap["totalCalls"].(int64)
		minuteCalls, _ = dmap["minuteCalls"].(int64)

		storedLastAccessed, _ := dmap["lastAccessed"].(int64)
		timeElapsed := lastAccessed - storedLastAccessed
		if timeElapsed < int64(60) {
			if minuteCalls >= callsPerMin {
				return &tooManyRequestsError{60 - int(timeElapsed)}
			} else {
				lastAccessed, _ = dmap["lastAccessed"].(int64)
				totalCalls++
				minuteCalls++
			}
		} else {
			// minute calls resets to 1
			minuteCalls = 1
			totalCalls++
		}
	}

	toWrite := map[string]interface{}{
		"username":     username,
		"totalCalls":   totalCalls,
		"minuteCalls":  minuteCalls,
		"lastAccessed": lastAccessed,
	}

	userDoc.Set(ctx, toWrite)

	return nil
}

// func firestoreInit() {
// 	// Get a Firestore client.
// 	ctx = context.Background()
// 	client := createClient()
// }
