package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

const MaxParticipants = 1000
const CurrentRound = 6

type Team struct {
	Rank     int    `json:"rank"`
	PrevRank int    `json:"prev_rank"`
	Name     string `json:"team"`
	Division string `json:"division"`
	Taken    bool
	TakenTwo bool
	MAC      int
}

type RawPreference struct {
	Team           string `json:"team"`
	Accept         string `json:"accept"`
	Challenge      string `json:"challenge"`
	PrevChallenged string `json:"prev_challenged"`
	LastResortPref string `json:"last_resort"`
	First          string `json:"first"`
	Second         string `json:"second"`
	Third          string `json:"third"`
}

type LastResortChallenge int

const (
	None LastResortChallenge = iota
	MinRank
	MaxRank
	Any
)

type ProcessedPreference struct {
	Team           *Team
	Accept         bool
	Challenge      bool
	PrevChallenged *Team
	LastResortPref LastResortChallenge
	First          string
	Second         string
	Third          string
}

type Challenge struct {
	ValidMatch     bool
	Round          int
	MatchCode      rune
	Challenger     *Team
	ChallengerRank int
	Defender       *Team
	DefenderRank   int
}

func loadTeams() map[string]*Team {
	// Get JSON file
	t, err := ioutil.ReadFile("teams.json")
	if err != nil {
		log.Fatal(err)
	}

	// Decode JSON
	var rawteams []*Team
	if err := json.Unmarshal(t, &rawteams); err != nil {
		log.Fatal(err)
	}

	teams := make(map[string]*Team)

	for _, team := range rawteams {
		team.Taken = false
		switch team.Division {
		case "X":
			team.MAC = team.Rank + 2
		case "S+":
			team.MAC = team.Rank + 3
		case "S":
			team.MAC = team.Rank + 4
		case "A":
			team.MAC = MaxParticipants
		}
		teams[team.Name] = team
	}

	return teams
}

func loadPrefs(teams map[string]*Team) map[string]*ProcessedPreference {
	p, err := ioutil.ReadFile("prefs.json")
	if err != nil {
		log.Fatal(err)
	}

	var raw_prefs []RawPreference
	if err := json.Unmarshal(p, &raw_prefs); err != nil {
		log.Fatal(err)
	}

	prefs := make(map[string]*ProcessedPreference)

	for _, raw_pref := range raw_prefs {
		var pref ProcessedPreference

		pref.Team = teams[raw_pref.Team]
		pref.PrevChallenged = teams[raw_pref.PrevChallenged]
		pref.First = raw_pref.First
		pref.Second = raw_pref.Second
		pref.Third = raw_pref.Third

		switch raw_pref.Accept {
		case "受け付ける":
			pref.Accept = true
		case "受け付けない":
			pref.Accept = false
		}

		if pref.Accept == false {
			teams[pref.Team.Name].Taken = true
			teams[pref.Team.Name].TakenTwo = true
		}

		switch raw_pref.Challenge {
		case "行う":
			pref.Challenge = true
		case "行わない":
			pref.Challenge = false
		}

		switch raw_pref.LastResortPref {
		case "どこにもチャレンジしない":
			pref.LastResortPref = None
		case "チャレンジ可能な範囲で一番順位の低いチームにチャレンジする":
			pref.LastResortPref = MinRank
		case "チャレンジ可能な範囲で一番順位の高いチームにチャレンジする":
			pref.LastResortPref = MaxRank
		case "自分より上位のチームならどこでもいいからチャレンジする":
			pref.LastResortPref = Any
		}

		prefs[raw_pref.Team] = &pref
	}
	return prefs
}

func getMinRank(teams map[string]*Team, sortedTeams []string, prefs map[string]*ProcessedPreference, challenge *Challenge) {
	challengerRank := challenge.Challenger.Rank
	fmt.Println("Trying to find an opponent. Challenger rank is ", challengerRank)

	for i := challengerRank - 1; i > 0; i-- {
		team := sortedTeams[i]
		fmt.Println("Checking if the following team is good:", team)
		if teams[team].MAC < challengerRank {
			fmt.Println("No, ranking too high. No valid match for", challenge.Challenger.Name)
			challenge.ValidMatch = false
			break
		}
		if validMatch(challenge.Challenger.Name, team, teams, prefs) == false {
			fmt.Println("Invalid match.")
		} else {
			fmt.Println("Minimum rank opponent available.")
			takeTeam(challenge.Challenger.Name, team, challenge, teams)
			break
		}
	}
}

func getMaxRank(teams map[string]*Team, sortedTeams []string, prefs map[string]*ProcessedPreference, challenge *Challenge) {
	challengerRank := challenge.Challenger.Rank
	fmt.Println("Trying to find an opponent. Challenger rank is ", challengerRank)

	for i := 1; i < challengerRank; i++ {
		team := sortedTeams[i]
		fmt.Println("Checking if the following team is good:", team)
		if teams[team].MAC < challengerRank {
			fmt.Println("No, ranking too high")
		} else if validMatch(challenge.Challenger.Name, team, teams, prefs) == false {
			fmt.Println("Invalid match.")
		} else if validMatch(challenge.Challenger.Name, team, teams, prefs) == true {
			fmt.Println("Maximum rank opponent available.")
			takeTeam(challenge.Challenger.Name, team, challenge, teams)
			break
		} else if i == challengerRank+1 {
			fmt.Println("No valid match for", challenge.Challenger.Name)
			challenge.ValidMatch = false
		}
	}
}

func checkTaken(team string, teams map[string]*Team) bool {
	// If not 1st, then just look at Taken
	if teams[team] != nil {
		if teams[team].Rank > 1 {
			return teams[team].Taken
		} else {
			return teams[team].Taken && teams[team].TakenTwo
		}
	}
	return true
}

func validMatch(challenger string, defender string, teams map[string]*Team, prefs map[string]*ProcessedPreference) bool {
	// Do these teams exist?
	if teams[challenger] == nil {
		return false
	}
	if teams[defender] == nil {
		return false
	}
	// Is the defender accepting matches?
	if prefs[defender].Accept == false {
		return false
	}
	// Did the challenger challenge defender in the previous round?
	if prefs[challenger].PrevChallenged != nil {
		if prefs[challenger].PrevChallenged.Name == teams[defender].Name {
			return false
		}
	}
	// Is the defender team taken?
	if checkTaken(defender, teams) == true {
		return false
	}
	// Is the defender's rank too high to be challenged?
	if teams[defender].MAC < teams[challenger].Rank {
		return false
	}

	return true
}

func takeTeam(challenger string, defender string, challenge *Challenge, teams map[string]*Team) {
	challenge.Defender = teams[defender]
	challenge.DefenderRank = teams[defender].Rank
	challenge.ValidMatch = true

	if teams[defender].Rank == 1 {
		if teams[defender].Taken == false {
			teams[defender].Taken = true
		} else {
			teams[defender].TakenTwo = true
		}
	} else {
		teams[defender].Taken = true
	}
	fmt.Println("Challenge accepted: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
}

func resolveChallenges(teams map[string]*Team, prefs map[string]*ProcessedPreference) (map[string]*Challenge, []string) {
	challenges := make(map[string]*Challenge)

	// 1. Sort teams by priority

	sortedTeams := make([]string, len(teams)+1)
	var newTeams []string
	var deferredTeams []string
	for team, info := range teams {
		if info.Rank > 0 {
			sortedTeams[info.Rank] = team
		} else {
			newTeams = append(newTeams, team)
		}
	}
	descSortedTeams := make([]string, len(sortedTeams))
	i := len(sortedTeams) - 1
	for _, value := range sortedTeams {
		descSortedTeams[i] = value
		i -= 1
	}
	for _, team := range newTeams {
		sortedTeams = append(sortedTeams, team)
		descSortedTeams = append(descSortedTeams, team)
	}
	fmt.Println(descSortedTeams)

	// 2. Give challenges to teams based on priorities

	for _, challenger := range descSortedTeams {
		if challenger != "" && prefs[challenger] != nil {
			var challenge Challenge
			challenge.Challenger = teams[challenger]
			challenge.ChallengerRank = teams[challenger].Rank
			challenge.Round = CurrentRound

			pref := prefs[challenger]

			if validMatch(challenger, pref.First, teams, prefs) {
				fmt.Println("First preference available for", challenger)
				takeTeam(challenger, pref.First, &challenge, teams)
			} else if validMatch(challenger, pref.Second, teams, prefs) {
				fmt.Println("Second preference available for", challenger)
				takeTeam(challenger, pref.Second, &challenge, teams)
			} else if validMatch(challenger, pref.Third, teams, prefs) {
				fmt.Println("Third preference available for", challenger)
				takeTeam(challenger, pref.Third, &challenge, teams)
			} else {
				fmt.Println("No preference available, checking last resort for", challenger)
				// Check for max or min
				switch pref.LastResortPref {
				case None:
					challenge.ValidMatch = false
					fmt.Println("No valid match for ", challenge.Challenger.Name)
				case MinRank:
					// Get the available challengeable team with minimum rank
					fmt.Println("Min rank opponent preferred.")
					getMinRank(teams, sortedTeams, prefs, &challenge)
				case MaxRank:
					fmt.Println("Max rank opponent preferred.")
					getMaxRank(teams, sortedTeams, prefs, &challenge)
					// Get the available challengeable team with maximum rank
				case Any:
					fmt.Println("Willing to challenge anyone.")
					deferredTeams = append(deferredTeams, challenger)
				}
			}

			if challenge.ValidMatch == true {
				challenges[challenger] = &challenge
			}
		}
	}

	// 3. Give challenges to deferred teams

	for _, challenger := range deferredTeams {
		if challenger != "" {
			fmt.Println("Checking opponents for", challenger)
			var challenge Challenge
			challenge.Challenger = teams[challenger]
			challenge.ChallengerRank = teams[challenger].Rank
			challenge.Round = CurrentRound

			for i := challenge.Challenger.Rank - 1; i > 0; i-- {
				team := sortedTeams[i]
				fmt.Println("Checking if the following team is good:", team)
				if validMatch(challenger, team, teams, prefs) == false {
					fmt.Println("Invalid match.")
				} else {
					takeTeam(challenger, team, &challenge, teams)
					break
				}
				if i == 1 {
					fmt.Println("No valid match for", challenger)
					challenge.ValidMatch = false
				}
			}
			if challenge.ValidMatch == true {
				challenges[challenger] = &challenge
			}
		}
	}

	// 4. Give MatchCodes accordingly

	code := 'A'

	for _, value := range sortedTeams {
		if value != "" && challenges[value] != nil {
			if challenges[value].ValidMatch == true {
				challenges[value].MatchCode = code
				code++
				if code == 'I' {
					code++
				}
			}
		}
	}

	for _, value := range newTeams {
		if value != "" && challenges[value] != nil {
			if challenges[value].ValidMatch == true {
				challenges[value].MatchCode = code
				code++
				if code == 'I' {
					code++
				}
			}
		}
	}

	return challenges, sortedTeams
}

func main() {

	// initialize teams and prefs
	teams := loadTeams()
	prefs := loadPrefs(teams)

	// resolve challenges based on teams and prefs
	challenges, sortedTeams := resolveChallenges(teams, prefs)

	for _, challenger := range sortedTeams {
		challenge := challenges[challenger]
		if challenge != nil {
			if challenge.ValidMatch {
				fmt.Println(challenge.Round, string(challenge.MatchCode), challenge.ChallengerRank, "位", challenge.Challenger.Name, "vs", challenge.DefenderRank, "位", challenge.Defender.Name)
			}
		}
	}
}
