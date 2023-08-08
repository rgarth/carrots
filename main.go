package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/slack-go/slack"
	"github.com/tkanos/gonfig"
)

// Configuration values that can be set
type configuration struct {
	Emoji         string
	Plural        string
	DBHost        string
	DBPort        int
	DBName        string
	DBUser        string
	DBPass        string
	SlackToken    string
	Limit         int
	PerUserLimit  int
	KudosResponse []string
	HelpResponse  []string
	SelfResponse  []string
}

type userStats struct {
	id       string
	sent     int
	received int
}

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("missing required environment variable " + name)
	}
	return v
}

func verifyRecipients(configuration *configuration, rtm *slack.RTM, ev *slack.MessageEvent, recipients []string, botUserID string) ([]string, error) {
	// Check is recipients re valid
	var verified []string
	for _, s := range recipients {
		recipient := strings.Replace(s, "@", "", -1)
		recipient = strings.ToUpper(recipient)

		// Can't give yourself points
		if recipient == ev.User {
			if len(configuration.SelfResponse) > 0 {
				return nil, errors.New(configuration.SelfResponse[rand.Intn(len(configuration.SelfResponse))])
			}
			return nil, errors.New("No patting yourself on the back")
		}
		// DO NOT FEED THE DONKEY
		if recipient == botUserID {
			return nil, errors.New("PLEASE DO NOT FEED THE DONKEY")
		}
		// Can only thank real people. Not rubber ducks.
		_, err := rtm.GetUserInfo(recipient)

		if err != nil {
			errResp := fmt.Sprintf("Who?")
			return nil, errors.New(errResp)
		}

		verified = append(verified, recipient)
	}
	return verified, nil
}

func verifyMonth(monthStr string) string {
	monthStr = strings.ToLower(monthStr)
	monthSlice := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}
	for _, s := range monthSlice {
		if monthStr == s {
			return monthStr
		}
	}
	return strings.ToLower(time.Now().Month().String())
}

func storeKudos(configuration *configuration, sender string, recipients []string, count int) error {
	dbConnectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		configuration.DBUser,
		configuration.DBPass,
		configuration.DBHost,
		configuration.DBPort,
		configuration.DBName)
	db, err := sql.Open("mysql", dbConnectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	insertStr := "INSERT INTO kudos_log(id, timestamp, sender, recipient) VALUES "
	insertRows := []string{}
	for i := 0; i < count; i++ {
		for _, recipient := range recipients {
			row := `
(0, NOW(), '` + sender + `', '` + recipient + `')`
			insertRows = append(insertRows, row)
		}
	}
	insertQuery := insertStr + strings.Join(insertRows, ",")

	insert, err := db.Query(insertQuery)
	if err != nil {
		return err
	}
	defer insert.Close()
	log.Println("Kudos saved in db")
	return nil
}

func getStats(configuration *configuration, user string, monthStr string) (userStats, error) {
	var stats userStats
	dbConnectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		configuration.DBUser,
		configuration.DBPass,
		configuration.DBHost,
		configuration.DBPort,
		configuration.DBName)
	db, dbErr := sql.Open("mysql", dbConnectionString)
	if dbErr != nil {
		return stats, dbErr
	}
	defer db.Close()
	sent := `select COUNT(sender) FROM kudos_log WHERE MONTHNAME(timestamp) = '` + monthStr +
		`' AND timestamp > DATE_SUB(DATE_FORMAT(NOW(), '%Y-%m-01'), INTERVAL 11 MONTH)` +
		` AND TRIM(sender) = '` + user + `'`
	sentQuery, sentErr := db.Query(sent)
	if sentErr != nil {
		return stats, sentErr
	}
	defer sentQuery.Close()
	for sentQuery.Next() {
		sentErr := sentQuery.Scan(&stats.sent)
		if sentErr != nil {
			return stats, sentErr
		}
	}
	rcvd := `select COUNT(recipient) FROM kudos_log WHERE MONTHNAME(timestamp) = '` + monthStr +
		`' AND timestamp > DATE_SUB(DATE_FORMAT(NOW(), '%Y-%m-01'), INTERVAL 11 MONTH)` +
		` AND TRIM(recipient) = '` + user + `'`
	rcvQuery, rcvErr := db.Query(rcvd)
	if rcvErr != nil {
		return stats, rcvErr
	}
	defer rcvQuery.Close()
	for rcvQuery.Next() {
		rcvErr := rcvQuery.Scan(&stats.received)
		if rcvErr != nil {
			return stats, rcvErr
		}
	}
	return stats, nil
}

func getLeaderboard(configuration *configuration, monthStr string) ([]userStats, userStats, error) {
	var topsender userStats
	var leaderboard []userStats
	dbConnectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		configuration.DBUser,
		configuration.DBPass,
		configuration.DBHost,
		configuration.DBPort,
		configuration.DBName)
	db, err := sql.Open("mysql", dbConnectionString)
	if err != nil {
		return nil, topsender, err
	}
	defer db.Close()

	// Leaderboard query
	queryStr := `select recipient,COUNT(DISTINCT(id)) from kudos_log WHERE MONTHNAME(timestamp) = '` + monthStr +
		`' AND timestamp > DATE_SUB(DATE_FORMAT(NOW(), '%Y-%m-01'), INTERVAL 11 MONTH)` +
		` GROUP BY recipient ORDER BY COUNT(DISTINCT(id)) DESC LIMIT 10`
	leaderboardQuery, err := db.Query(queryStr)
	if err != nil {
		return nil, topsender, err
	}
	defer leaderboardQuery.Close()
	for leaderboardQuery.Next() {
		var current userStats
		leaderboardQuery.Scan(&current.id, &current.received)
		if err != nil {
			return nil, current, err
		}
		leaderboard = append(leaderboard, current)
	}
	// topsender query
	queryStr = `select sender,COUNT(DISTINCT(id)) from kudos_log WHERE MONTHNAME(timestamp) = '` + monthStr +
		`' AND timestamp > DATE_SUB(DATE_FORMAT(NOW(), '%Y-%m-01'),  INTERVAL 11 MONTH)` +
		` GROUP BY sender ORDER BY COUNT(DISTINCT(id)) DESC LIMIT 1`
	topsenderQuery, err := db.Query(queryStr)
	if err != nil {
		return nil, topsender, err
	}
	defer topsenderQuery.Close()
	for topsenderQuery.Next() {
		topsenderQuery.Scan(&topsender.id, &topsender.sent)
		if err != nil {
			return nil, topsender, err
		}
	}
	return leaderboard, topsender, nil
}

func main() {
	configuration := configuration{}

	// default emoji overwritten by config file
	configuration.Emoji = "carrot"
	configuration.Plural = "carrots"
	configuration.Limit = -1        // default here is no monthly kudos limit
	configuration.PerUserLimit = -1 // default is no limit to per user in action
	err := gonfig.GetConf("./carrots.json", &configuration)
	if err != nil {
		panic(err)
	}

	configuration.SlackToken = getenv("SLACKTOKEN")
	configuration.DBPass = getenv("MYSQLPASS")
	api := slack.New(configuration.SlackToken)
	rtm := api.NewRTM()

	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.MessageEvent:
				info := rtm.GetInfo()

				text := ev.Text
				text = strings.TrimSpace(text)
				text = strings.ToLower(text)
				sender, _ := rtm.GetUserInfo(ev.User)

				carrotsMatch := regexp.MustCompile(fmt.Sprintf(":%s:", configuration.Emoji))
				usersMatch := regexp.MustCompile("@([A-Za-z]+[A-Za-z0-9-_]+)")
				cmdMatch := regexp.MustCompile(fmt.Sprintf(`^<@%s> *(.*)$`, strings.ToLower(info.User.ID)))

				carrots := carrotsMatch.FindAllStringIndex(text, -1)
				recipients := usersMatch.FindAllString(text, -1)
				atCmd := cmdMatch.FindStringSubmatch(text)

				if ev.User != info.User.ID && len(carrots) > 0 && len(recipients) > 0 {
					// verify recipients are valid
					strings.ToLower(info.User.ID)
					verified, err := verifyRecipients(&configuration, rtm, ev, recipients, info.User.ID)

					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(err.Error(), ev.Channel))
					} else {
						//Genuine Kudos!
						//Check if we have the budget
						haveBudget := true
						var lookupError error = nil
						if configuration.Limit != -1 {
							var mystats userStats
							mystats, lookupError = getStats(&configuration, sender.ID, time.Now().Month().String())
							if lookupError != nil {
								rtm.SendMessage(rtm.NewOutgoingMessage(
									fmt.Sprintf("Thanks for sharing the :%s:, unfortunately I could not find my %s store",
										configuration.Emoji, configuration.Plural),
									ev.Channel))
							}
							if mystats.sent+(len(carrots)*len(verified)) > configuration.Limit {
								haveBudget = false
							}
						}
						if configuration.PerUserLimit != -1 {
							if len(carrots) > configuration.PerUserLimit {
								haveBudget = false
							}
						}
						if haveBudget == true && lookupError == nil {
							//save to db
							err := storeKudos(&configuration, sender.ID, verified, len(carrots))
							if err != nil {
								rtm.SendMessage(rtm.NewOutgoingMessage(
									fmt.Sprintf("Thanks for sharing the :%s:, unfortunately I had a problem saving them", configuration.Emoji),
									ev.Channel))
							} else {

								// Acknowledge the carrots
								var botResp string
								log.Printf("%s sent %d %s to %d verified users\n", sender.Profile.RealName, len(carrots), configuration.Plural, len(verified))
								if len(configuration.KudosResponse) > 0 {
									botResp = configuration.KudosResponse[rand.Intn(len(configuration.KudosResponse))]
								}
								for i := 0; i < (len(carrots) * len(verified)); i++ {
									botResp = fmt.Sprintf("%s :%s:", botResp, configuration.Emoji)
								}
								botResp = fmt.Sprintf("%s :heart:", botResp)
								text := slack.MsgOptionText(botResp, false)
								user := slack.MsgOptionAsUser(true)
								rtm.PostEphemeral(ev.Channel, sender.ID, text, user)
								rtm.AddReaction("heart", slack.ItemRef{ev.Channel, ev.Timestamp, "", ""})
							}
						} else if lookupError == nil {
							if len(carrots) > configuration.PerUserLimit {
								rtm.SendMessage(rtm.NewOutgoingMessage(
									fmt.Sprintf("Thanks for sharing the :%s:, unfortunately you can't send more than %d in a message",
										configuration.Emoji, configuration.PerUserLimit),
									ev.Channel))
							} else {
								rtm.SendMessage(rtm.NewOutgoingMessage(
									fmt.Sprintf("Thanks for sharing the :%s:, unfortunately you can't send more than %d in a month",
										configuration.Emoji, configuration.Limit),
									ev.Channel))
							}
						}

					}

				} else if atCmd != nil {
					cmdStr := atCmd[len(atCmd)-1]
					ladderMatch := regexp.MustCompile(`^ladder *([A-Za-z]*)$`)
					ladderCmd := ladderMatch.FindStringSubmatch(cmdStr)
					if cmdStr == "me" {
						var respStr string
						log.Println(fmt.Sprintf("Looking up stats for %s", sender.RealName))
						stats, err := getStats(&configuration, sender.ID, time.Now().Month().String())
						if err != nil {
							log.Println(err)
							respStr = fmt.Sprintf("Sorry, I encountered a problem and couldn't look up your stats")
						} else {
							respStr = fmt.Sprintf("Hey, so far in %s, you have given *%d* %s, and received *%d*", time.Now().Month().String(), stats.sent, configuration.Plural, stats.received)
							if configuration.Limit != -1 {
								respStr = respStr + fmt.Sprintf("\nYou can send a total of *%d* :%s: per month",
									configuration.Limit, configuration.Emoji)
							}
						}
						text := slack.MsgOptionText(respStr, false)
						user := slack.MsgOptionAsUser(true)
						rtm.PostEphemeral(ev.Channel, sender.ID, text, user)

					} else if ladderCmd != nil {
						monthStr := verifyMonth(ladderCmd[len(ladderCmd)-1])
						log.Printf("Looking up leaderboard for %s", strings.Title(monthStr))
						leaderboard, topsender, err := getLeaderboard(&configuration, monthStr)
						var respStr string
						if err != nil {
							log.Println(err)
							respStr = fmt.Sprintf("Sorry, I encountered a problem and couldn't look up the current leaderboard")
						} else {
							if len(leaderboard) > 0 {
								respStr = fmt.Sprintf("The standings for %s:", strings.Title(monthStr))
								for i, ranked := range leaderboard {
									rankedUser, err := rtm.GetUserInfo(ranked.id)
									realName := "Unkown"
									if err == nil {
										realName = rankedUser.RealName
									}
									respStr = respStr + fmt.Sprintf("\n> %d. *%s* received %d :%s:",
										i+1, realName, ranked.received, configuration.Emoji)
								}
								user, err := rtm.GetUserInfo(topsender.id)
								if err == nil {
									respStr = respStr + fmt.Sprintf("\n> \n> *%s* gave the most! %d :%s:",
										user.RealName, topsender.sent, configuration.Emoji)
								}
							} else {
								respStr = fmt.Sprintf("In %s, no :%s: were given :cry:", strings.Title(monthStr), configuration.Emoji)
							}
						}
						rtm.SendMessage(rtm.NewOutgoingMessage(respStr, ev.Channel))

					} else {
						user := slack.MsgOptionAsUser(true)
						helpStr := fmt.Sprintf("*Send %s to your friends:*", configuration.Plural) +
							fmt.Sprintf("\n>Hey @shrek, I like you, have a :%s:",
								configuration.Emoji) +
							fmt.Sprintf("\n>:%s: @shrek @fiona", configuration.Emoji) +
							fmt.Sprintf("\n>I like that @boulder, it's a nice boulder :%s: :%s:",
								configuration.Emoji, configuration.Emoji) +
							fmt.Sprintf("\n*Other stuff:*") +
							fmt.Sprintf("\n>`@%s me` Find out how many :%s: you have",
								info.User.Name, configuration.Emoji) +
							fmt.Sprintf("\n>`@%s ladder [month]` Find out who has the most :%s:",
								info.User.Name, configuration.Emoji) +
							fmt.Sprintf("\n>`@%s help` Print this message", info.User.Name)
						if configuration.Limit != -1 {
							helpStr = helpStr + fmt.Sprintf("\nYou can send a total of *%d* %s per month",
								configuration.Limit, configuration.Plural)
						}
						if len(configuration.HelpResponse) > 0 {
							helpStr = helpStr + fmt.Sprintf("\n%s",
								configuration.HelpResponse[rand.Intn(len(configuration.HelpResponse))])
						}
						text := slack.MsgOptionText(helpStr, false)
						rtm.PostEphemeral(ev.Channel, sender.ID, text, user)
					}
				}

			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				log.Printf("Invalid credentials")
				break Loop

			default:
				// Take no action
			}
		}
	}
}
