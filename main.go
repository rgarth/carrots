package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("missing required environment variable " + name)
	}
	return v
}

func verifyRecipients(rtm *slack.RTM, ev *slack.MessageEvent, recipients []string) ([]string, error) {
	// Check is recipients re valid
	var verified []string
	for i, s := range recipients {
		recipient := strings.Replace(s, "@", "", -1)
		recipient = strings.ToUpper(recipient)
		fmt.Println(i, recipient)

		// Can't give yourself points
		if recipient == ev.User {
			return nil, errors.New("no patting yourself on the back")
		}
		// Can only thank real people. Not rubber ducks.
		recipientData, err := rtm.GetUserInfo(recipient)

		if err != nil {
			errResp := fmt.Sprintf("can't find %s to give them any carrots", s)
			return nil, errors.New(errResp)
		}

		verified = append(verified, recipientData.Name)
	}
	return verified, nil
}

func main() {
	token := getenv("SLACKTOKEN")
	api := slack.New(token)
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

				carrotsMatch := regexp.MustCompile(":carrot:")
				usersMatch := regexp.MustCompile("@([A-Za-z]+[A-Za-z0-9-_]+)")
				carrots := carrotsMatch.FindAllStringIndex(text, -1)
				recipients := usersMatch.FindAllString(text, -1)

				if ev.User != info.User.ID && len(carrots) > 0 && len(recipients) > 0 {
					sender, _ := rtm.GetUserInfo(ev.User)
					fmt.Println(sender.Name)
					// verify recipients are valid
					verified, err := verifyRecipients(rtm, ev, recipients)

					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(err.Error(), ev.Channel))
						fmt.Println(err)
					} else {
						fmt.Println(verified)
						//Genuine Kudos!
						//save to db TDB
						botResp := ""
						for i := 0; i < (len(carrots) * len(verified)); i++ {
							botResp = fmt.Sprintf("%s :carrot:", botResp)
						}
						botResp = fmt.Sprintf("%s :heart:", botResp)
						rtm.SendMessage(rtm.NewOutgoingMessage(botResp, ev.Channel))
						println(botResp)
					}

				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// Take no action
			}
		}
	}
}
