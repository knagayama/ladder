package main

import (
	"cloud.google.com/go/firestore"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"log"
)

func saveTeams(teams map[string]*Team, client *firestore.Client, ctx context.Context) {
	for key, team := range teams {
		_, err := client.Collection("teams").Doc(key).Set(ctx, team)

		if err != nil {
			log.Fatalf("Failed adding teams: %v", err)
		}
	}

	iter := client.Collection("teams").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println(doc.Data())
	}
}

func savePrefs(prefs map[string]*ProcessedPreference, client *firestore.Client, ctx context.Context) {
	for key, pref := range prefs {
		_, err := client.Collection("prefs").Doc(key).Set(ctx, pref)

		if err != nil {
			log.Fatalf("Failed adding prefs: %v", err)
		}
	}

	iter := client.Collection("prefs").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println(doc.Data())
	}
}
