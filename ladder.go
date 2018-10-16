package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

const MaxParticipants = 1000
const CurrentRound = 3

type Team struct {
	Rank     int    `json:"rank"`
	PrevRank int    `json:"prev_rank"`
	Name     string `json:"team"`
	Division string `json:"division"`
	Taken    bool
	MAC      int
}

type RawPreference struct {
	Team           string `json:"Team"`
	Accept         string `json:"Accept"`
	Challenge      string `json:"Challenge"`
	PrevChallenged string `json:"PrevChallenged"`
	LastResortPref string `json:"LastResortPref"`
	First          string `json:"First"`
	Second         string `json:"Second"`
	Third          string `json:"Third"`
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
	MatchCode      string
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

func getMinRank(teams map[string]*Team, sortedTeams []string, challenge *Challenge) {
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
		if teams[team].Taken == true {
			fmt.Println("Team is taken.")
		} else if teams[team].Taken == false {
			challenge.Defender = teams[team]
			challenge.DefenderRank = challenge.Defender.Rank
			challenge.ValidMatch = true
			teams[team].Taken = true
			fmt.Println("Minimum rank opponent available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
			break
		}
	}
}

func getMaxRank(teams map[string]*Team, sortedTeams []string, challenge *Challenge) {
	challengerRank := challenge.Challenger.Rank
	fmt.Println("Trying to find an opponent. Challenger rank is ", challengerRank)

	for i := 1; i < challengerRank; i++ {
		team := sortedTeams[i]
		fmt.Println("Checking if the following team is good:", team)
		if teams[team].MAC < challengerRank {
			fmt.Println("No, ranking too high")
		} else if teams[team].Taken == true {
			fmt.Println("Team is taken.")
		} else if teams[team].Taken == false {
			challenge.Defender = teams[team]
			challenge.DefenderRank = challenge.Defender.Rank
			challenge.ValidMatch = true
			teams[team].Taken = true
			fmt.Println("Maximum rank opponent available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
			break
		} else if i == challengerRank+1 {
			fmt.Println("No valid match for", challenge.Challenger.Name)
			challenge.ValidMatch = false
		}
	}
}

func resolveChallenges(teams map[string]*Team, prefs map[string]*ProcessedPreference) map[string]*Challenge {
	challenges := make(map[string]*Challenge)

	// 1. Give priorities to teams
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

	// 2. Give challenges to teams based on priorities

	for _, value := range descSortedTeams {
		if value != "" {
			var challenge Challenge
			challenge.Challenger = teams[value]
			challenge.Round = CurrentRound

			pref := prefs[value]

			if pref.First != "" && teams[pref.First].Taken == false {
				challenge.Defender = teams[pref.First]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.First].Taken = true
				fmt.Println("First preference available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
			} else if pref.Second != "" && teams[pref.Second].Taken == false {
				challenge.Defender = teams[pref.Second]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.Second].Taken = true
				fmt.Println("Second preference available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
			} else if pref.Third != "" && teams[pref.Third].Taken == false {
				challenge.Defender = teams[pref.Third]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.Third].Taken = true
				fmt.Println("Third preference available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
			} else {
				fmt.Println("No preference available, checking last resort for", value)
				// Check for max or min
				switch pref.LastResortPref {
				case None:
					challenge.ValidMatch = false
					fmt.Println("No valid match for ", challenge.Challenger.Name)
				case MinRank:
					// Get the available challengeable team with minimum rank
					fmt.Println("Min rank opponent preferred.")
					getMinRank(teams, sortedTeams, &challenge)
				case MaxRank:
					fmt.Println("Max rank opponent preferred.")
					getMaxRank(teams, sortedTeams, &challenge)
					// Get the available challengeable team with maximum rank
				case Any:
					fmt.Println("Willing to challenge anyone.")
					deferredTeams = append(deferredTeams, value)
				}
			}

			if challenge.ValidMatch == true {
				challenges[value] = &challenge
			}
		}
	}

	// 3. Give challenges to new teams

	for _, value := range newTeams {
		if value != "" {
			var challenge Challenge
			challenge.Challenger = teams[value]
			challenge.Round = CurrentRound

			pref := prefs[value]

			if pref.First != "" && teams[pref.First].Taken == false {
				challenge.Defender = teams[pref.First]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.First].Taken = true
				fmt.Println("First preference available: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Name)
			} else if pref.Second != "" && teams[pref.Second].Taken == false {
				challenge.Defender = teams[pref.Second]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.Second].Taken = true
				fmt.Println("Second preference available: ", challenge.Challenger.Name, "vs", challenge.Defender.Name)
			} else if pref.Third != "" && teams[pref.Third].Taken == false {
				challenge.Defender = teams[pref.Third]
				challenge.DefenderRank = challenge.Defender.Rank
				challenge.ValidMatch = true
				teams[pref.Third].Taken = true
				fmt.Println("Third preference available: ", challenge.Challenger.Name, "vs", challenge.Defender.Name)
			} else {
				fmt.Println("No preference available, deferring for ", value)
				deferredTeams = append(deferredTeams, value)
			}

			if challenge.ValidMatch == true {
				challenges[value] = &challenge
			}
		}
	}

	// 4. Give challenges to deferred teams

	for _, value := range deferredTeams {
		if value != "" {
			fmt.Println("Checking opponents for", value)
			var challenge Challenge
			challenge.Challenger = teams[value]
			challenge.Round = CurrentRound

			for i := challenge.Challenger.Rank - 1; i > 0; i-- {
				team := sortedTeams[i]
				fmt.Println("Checking if the following team is good:", team)
				if teams[team].Taken == true {
					fmt.Println("Team is taken.")
				} else if teams[team].Taken == false {
					challenge.Defender = teams[team]
					challenge.DefenderRank = challenge.Defender.Rank
					challenge.ValidMatch = true
					teams[team].Taken = true
					fmt.Println("Available opponent found: ", challenge.Challenger.Rank, "位", challenge.Challenger.Name, "vs", challenge.Defender.Rank, "位", challenge.Defender.Name)
					break
				}
				if i == 1 {
					fmt.Println("No valid match for", value)
					challenge.ValidMatch = false
				}
			}
		}
	}

	return challenges
}

func main() {

	// initialize teams and prefs
	teams := loadTeams()
	prefs := loadPrefs(teams)

	// resolve challenges based on teams and prefs
	challenges := resolveChallenges(teams, prefs)

	for _, value := range challenges {
		if value.ValidMatch {
			fmt.Println(value.Challenger.Name, "vs", value.Defender.Name)
		}
	}
}
