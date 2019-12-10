package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTeamsFromSpreadsheet() map[string]*Team {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// Get teams from a preformatted sheet in the challenge form.
	spreadsheetId := "1zEw8Eb2WGzY8nZt_6B5rL9v_6PUW7CUBusvoqccrayQ"
	readRange := "teams!A2:E"
	valueRenderOption := "UNFORMATTED_VALUE"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).ValueRenderOption(valueRenderOption).Do()

	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	teams := make(map[string]*Team)

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range resp.Values {
			// The spreadsheet is ordered as prev_rank, rank, new, division, team
			fmt.Println(row)
			var team Team
			team.PrevRank = int(row[0].(float64))
			team.Rank = int(row[1].(float64))
			team.New = row[2].(bool)
			team.Division = row[3].(string)
			team.Name = row[4].(string)
			switch team.Division {
			case "X":
				team.MAC = team.Rank + 2
			case "S+":
				team.MAC = team.Rank + 3
			case "S":
				team.MAC = team.Rank + 4
			case "A+":
				team.MAC = team.Rank + 5
			case "A":
				team.MAC = MaxParticipants
			}
			team.Taken = false
			fmt.Println(team)
			teams[team.Name] = &team
		}
	}
	return teams
}

func getPrefsFromSpreadsheet() []RawPreference {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// Get prefs from a preformatted sheet in the challenge form.
	spreadsheetId := "1zEw8Eb2WGzY8nZt_6B5rL9v_6PUW7CUBusvoqccrayQ"
	readRange := "prefs!A2:I"
	valueRenderOption := "FORMATTED_VALUE"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).ValueRenderOption(valueRenderOption).Do()

	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	var raw_prefs []RawPreference

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range resp.Values {
			// The spreadsheet is formatted as accept, challenge, current_rank, last_resort,
			// prev_challenged, first, second, third, team.
			fmt.Println(row)
			var pref RawPreference
			pref.Accept = row[0].(string)
			pref.Challenge = row[1].(string)
			pref.LastResortPref = row[3].(string)
			pref.PrevChallenged = row[4].(string)
			pref.First = row[5].(string)
			pref.Second = row[6].(string)
			pref.Third = row[7].(string)
			pref.Team = row[8].(string)
			fmt.Println(pref)
			raw_prefs = append(raw_prefs, pref)
		}
	}
	return raw_prefs
}
