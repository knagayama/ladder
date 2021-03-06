package main

import (
	"flag"
	"fmt"
)

const MaxParticipants = 1000

type Round struct {
	Teams     map[string]*Team
	NewTeams  []string
	AscOrder  []string
	DescOrder []string
	Prefs     map[string]*ProcessedPreference
	Chals     map[string]*Challenge
	Current   int
}

type Team struct {
	Rank     int    `json:"rank"`
	PrevRank int    `json:"prev_rank"`
	Name     string `json:"team"`
	Division string `json:"division"`
	New      bool   `json:"new"`
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
	Team           string
	Accept         bool
	Challenge      bool
	PrevChallenged string
	LastResortPref LastResortChallenge
	First          string
	Second         string
	Third          string
}

type Challenge struct {
	ValidMatch     bool
	Round          int
	MatchCode      int
	Challenger     string
	ChallengerRank int
	Defender       string
	DefenderRank   int
}

func (round *Round) initRound(currentRound int) {
	// 1. Load teams.

	teams := getTeamsFromSpreadsheet()
	fmt.Println("Loaded teams:", len(teams))
	round.Teams = teams

	// 2. Sort teams by priority

	sortedTeams := make([]string, len(teams)+1)
	var newTeams []string
	for team, info := range teams {
		if info.New == false {
			sortedTeams[info.Rank] = team
		} else {
			newTeams = append(newTeams, team)
		}
	}
	descSortedTeams := make([]string, len(sortedTeams))
	i := len(sortedTeams) - 1
	for _, value := range sortedTeams {
		descSortedTeams[i] = value
		i--
	}

	for _, team := range newTeams {
		sortedTeams = append(sortedTeams, team)
	}

	round.NewTeams = newTeams
	round.AscOrder = sortedTeams
	round.DescOrder = descSortedTeams

	// 3. Load preferences.
	rawPrefs := getPrefsFromSpreadsheet()

	prefs := make(map[string]*ProcessedPreference)

	for _, rawPref := range rawPrefs {
		var pref ProcessedPreference

		pref.Team = rawPref.Team
		pref.PrevChallenged = rawPref.PrevChallenged
		pref.First = rawPref.First
		pref.Second = rawPref.Second
		pref.Third = rawPref.Third

		switch rawPref.Accept {
		case "受け付ける":
			pref.Accept = true
		case "受け付けない":
			pref.Accept = false
		}

		if pref.Accept == false {
			round.Teams[pref.Team].Taken = true
			round.Teams[pref.Team].TakenTwo = true
		}

		switch rawPref.Challenge {
		case "行う":
			pref.Challenge = true
		case "行わない":
			pref.Challenge = false
		}

		switch rawPref.LastResortPref {
		case "どこにもチャレンジしない":
			pref.LastResortPref = None
		case "チャレンジ可能な範囲で一番順位の低いチームにチャレンジする":
			pref.LastResortPref = MinRank
		case "チャレンジ可能な範囲で一番順位の高いチームにチャレンジする":
			pref.LastResortPref = MaxRank
		case "自分より上位のチームならどこでもいいからチャレンジする":
			pref.LastResortPref = Any
		}

		prefs[pref.Team] = &pref
	}

	fmt.Println("Loaded prefs:", len(prefs))
	round.Prefs = prefs

	round.Current = currentRound
}

func (round *Round) validateMatch(challenger string, defender string, ignoreMac bool) bool {
	fmt.Println("Validating", challenger, "vs", defender)
	teams := round.Teams
	prefs := round.Prefs

	// Do these teams exist?
	if teams[challenger] == nil {
		fmt.Println(challenger, "does not exist.")
		return false
	}
	if teams[defender] == nil {
		fmt.Println(defender, "does not exist.")
		return false
	}
	// Is the defender accepting matches?
	if prefs[defender].Accept == false {
		fmt.Println(defender, "is not accepting challenges.")
		return false
	}
	// Did the challenger challenge defender in the previous round?
	if prefs[challenger].PrevChallenged != "" {
		if prefs[challenger].PrevChallenged == teams[defender].Name {
			fmt.Println(challenger, "already challenged", defender, "last round.")
			return false
		}
	}
	// Is the defender team taken?
	if round.checkTaken(defender) == true {
		fmt.Println(defender, "is taken.")
		return false
	}
	// Is the challenger's rank lower than defender's rank?
	if teams[challenger].Rank < teams[defender].Rank {
		fmt.Println("Challenging", challenger, "rank is higher than defending", defender)
		return false
	}
	// Is the defender's rank too high to be challenged?
	if ignoreMac == false && teams[defender].MAC < teams[challenger].Rank {
		fmt.Println(defender, "rank is too high to be challenged.")
		return false
	}

	return true
}

func (round *Round) checkTaken(team string) bool {
	teams := round.Teams
	// If not 1st, then just look at Taken
	if teams[team] != nil {
		if teams[team].Rank > 1 {
			return teams[team].Taken
		}
		return teams[team].Taken && teams[team].TakenTwo
	}
	return true
}

func (round *Round) takeTeam(challenger string, defender string, challenge *Challenge) {
	teams := round.Teams

	challenge.Defender = defender
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
	fmt.Println("Challenge accepted: ", challenge.ChallengerRank, "位", challenge.Challenger, "vs", challenge.DefenderRank, "位", challenge.Defender)
}

func (round *Round) challengeMinRank(challenge *Challenge) {
	teams := round.Teams
	ascSortedTeams := round.AscOrder
	challengerRank := teams[challenge.Challenger].Rank
	fmt.Println("Trying to find an opponent. Challenger rank is ", challengerRank)

	for i := challengerRank - 1; i > 0; i-- {
		team := ascSortedTeams[i]
		fmt.Println("Checking if the following team is good:", team)
		if teams[team].MAC < challengerRank {
			fmt.Println("No, ranking too high. No valid match for", challenge.Challenger)
			challenge.ValidMatch = false
			break
		}
		if round.validateMatch(challenge.Challenger, team, false) == false {
			fmt.Println("Invalid match.")
		} else {
			fmt.Println("Minimum rank opponent available.")
			round.takeTeam(challenge.Challenger, team, challenge)
			break
		}
	}
}

func (round *Round) challengeMaxRank(challenge *Challenge) {
	teams := round.Teams
	ascSortedTeams := round.AscOrder
	challengerRank := teams[challenge.Challenger].Rank
	fmt.Println("Trying to find an opponent. Challenger rank is ", challengerRank)

	for i := 1; i < challengerRank; i++ {
		team := ascSortedTeams[i]
		fmt.Println("Checking if the following team is good:", team)
		if teams[team].MAC < challengerRank {
			fmt.Println("No, ranking too high")
		} else if round.validateMatch(challenge.Challenger, team, false) == false {
			fmt.Println("Invalid match.")
		} else if round.validateMatch(challenge.Challenger, team, false) == true {
			fmt.Println("Maximum rank opponent available.")
			round.takeTeam(challenge.Challenger, team, challenge)
			break
		} else if i == challengerRank+1 {
			fmt.Println("No valid match for", challenge.Challenger)
			challenge.ValidMatch = false
		}
	}
}

func (round *Round) generateChallenges(manualAssignLeftover bool) {
	challenges := make(map[string]*Challenge)
	teams := round.Teams
	prefs := round.Prefs
	descSortedTeams := round.DescOrder
	ascSortedTeams := round.AscOrder
	newTeams := round.NewTeams
	var deferredTeams []string

	// Give challanges to new teams

	for _, challenger := range newTeams {
		if challenger != "" && prefs[challenger] != nil && prefs[challenger].Challenge {
			var challenge Challenge
			challenge.Challenger = challenger
			challenge.ChallengerRank = teams[challenger].Rank
			challenge.Round = round.Current

			fmt.Println("Trying to give a match to", challenger)
			pref := prefs[challenger]

			if round.validateMatch(challenger, pref.First, true) {
				fmt.Println("First preference available for", challenger)
				round.takeTeam(challenger, pref.First, &challenge)
			} else if round.validateMatch(challenger, pref.Second, true) {
				fmt.Println("Second preference available for", challenger)
				round.takeTeam(challenger, pref.Second, &challenge)
			} else if round.validateMatch(challenger, pref.Third, true) {
				fmt.Println("Third preference available for", challenger)
				round.takeTeam(challenger, pref.Third, &challenge)
			} else {
				fmt.Println("No preference available, checking last resort for", challenger)
				// Check for max or min
				switch pref.LastResortPref {
				case None:
					challenge.ValidMatch = false
					fmt.Println("No valid match for ", challenge.Challenger)
				case MinRank:
					// Get the available challengeable team with minimum rank
					fmt.Println("Min rank opponent preferred.")
					round.challengeMinRank(&challenge)
				case MaxRank:
					fmt.Println("Max rank opponent preferred.")
					round.challengeMaxRank(&challenge)
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


	// Give challenges to teams based on priorities

	for _, challenger := range descSortedTeams {
		if challenger != "" && prefs[challenger] != nil && prefs[challenger].Challenge {
			var challenge Challenge
			challenge.Challenger = challenger
			challenge.ChallengerRank = teams[challenger].Rank
			challenge.Round = round.Current

			fmt.Println("Trying to give a match to", challenger)
			pref := prefs[challenger]

			if round.validateMatch(challenger, pref.First, false) {
				fmt.Println("First preference available for", challenger)
				round.takeTeam(challenger, pref.First, &challenge)
			} else if round.validateMatch(challenger, pref.Second, false) {
				fmt.Println("Second preference available for", challenger)
				round.takeTeam(challenger, pref.Second, &challenge)
			} else if round.validateMatch(challenger, pref.Third, false) {
				fmt.Println("Third preference available for", challenger)
				round.takeTeam(challenger, pref.Third, &challenge)
			} else {
				fmt.Println("No preference available, checking last resort for", challenger)
				// Check for max or min
				switch pref.LastResortPref {
				case None:
					challenge.ValidMatch = false
					fmt.Println("No valid match for ", challenge.Challenger)
				case MinRank:
					// Get the available challengeable team with minimum rank
					fmt.Println("Min rank opponent preferred.")
					round.challengeMinRank(&challenge)
				case MaxRank:
					fmt.Println("Max rank opponent preferred.")
					round.challengeMaxRank(&challenge)
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

	// Give challenges to deferred teams

	if !manualAssignLeftover {
		// Auto-assignment for deferred teams
		for _, challenger := range deferredTeams {
			if challenger != "" {
				fmt.Println("Checking opponents for", challenger)
				var challenge Challenge
				challenge.Challenger = challenger
				challenge.ChallengerRank = teams[challenger].Rank
				challenge.Round = round.Current

				for i := challenge.ChallengerRank - 1; i > 0; i-- {
					team := ascSortedTeams[i]
					fmt.Println("Checking if the following team is good:", team)
					if round.validateMatch(challenger, team, true) == false {
						fmt.Println("Invalid match.")
					} else {
						round.takeTeam(challenger, team, &challenge)
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
	} else {
		// Manual assign for deferred teams
		for _, challenger := range deferredTeams {
			fmt.Println("Manual assign needed for: ", challenger, "@", teams[challenger].Rank)
			for team := range teams {
				if round.Teams[team].Taken == false {
					fmt.Println(team, " is not taken @", round.Teams[team].Rank)
				}
				if round.Teams[team].Rank == 1 && round.Teams[team].TakenTwo == false {
					fmt.Println(team, " is not taken @", round.Teams[team].Rank)
				}
			}
			var challenge Challenge
			challenge.Challenger = challenger
			challenge.ChallengerRank = teams[challenger].Rank
			challenge.Round = round.Current
			fmt.Println("Choose team rank to assign for ", challenger, "@", teams[challenger].Rank)
			var i int
			fmt.Scanf("%d", &i)
			team := ascSortedTeams[i]
			if round.validateMatch(challenger, team, true) == false {
				fmt.Println("Invalid match.")
			} else {
				round.takeTeam(challenger, team, &challenge)
			}
			if challenge.ValidMatch == true {
				challenges[challenger] = &challenge
			}
		}
	}

	// Give MatchCodes accordingly

	code := 1

	for _, value := range ascSortedTeams {
		if value != "" && challenges[value] != nil {
			if challenges[value].ValidMatch == true {
				challenges[value].MatchCode = code
				code++
			}
		}
	}

	round.Chals = challenges
}

func (round *Round) printChallenges() {
	fmt.Println("==== ラウンド", round.Current, "全試合 ====")
	for _, challenger := range round.AscOrder {
		challenge := round.Chals[challenger]
		if challenge != nil && challenge.ValidMatch {
			if round.Teams[challenger].New {
				fmt.Printf("[%d-%02d] New! %s vs %02d位 %s\n", challenge.Round, challenge.MatchCode, challenge.Challenger, challenge.DefenderRank, challenge.Defender)
			} else {
				fmt.Printf("[%d-%02d] %02d位 %s vs %02d位 %s\n", challenge.Round, challenge.MatchCode, challenge.ChallengerRank, challenge.Challenger, challenge.DefenderRank, challenge.Defender)
			}
		}
	}
	fmt.Println("==== ラウンド", round.Current, "全試合csv ====")
	fmt.Println("id,挑戦側rank,挑戦側チーム名,防衛側rank,防衛側チーム名")
	for _, challenger := range round.AscOrder {
		challenge := round.Chals[challenger]
		if challenge != nil && challenge.ValidMatch {
			if round.Teams[challenger].New {
				fmt.Printf("[%d-%02d],New,%s,%02d,%s\n", challenge.Round, challenge.MatchCode, challenge.Challenger, challenge.DefenderRank, challenge.Defender)
			} else {
				fmt.Printf("[%d-%02d],%02d位,%s,%02d位,%s\n", challenge.Round, challenge.MatchCode, challenge.ChallengerRank, challenge.Challenger, challenge.DefenderRank, challenge.Defender)
			}
		}
	}
}

func main() {
	var round Round
	currentRound := flag.Int("round", 0, "Current round")
	manualAssignLeftover := flag.Bool("manual", false, "Manually assign leftovers")
	flag.Parse()
	round.initRound(*currentRound)
	round.generateChallenges(*manualAssignLeftover)
	round.printChallenges()
}
