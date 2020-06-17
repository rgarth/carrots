package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
	"github.com/tkanos/gonfig"
)

// Configuration values that can be set
type Configuration struct {
	Emoji  string
	Plural string
}

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("missing required environment variable " + name)
	}
	return v
}

func verifyRecipients(configuration *Configuration, rtm *slack.RTM, ev *slack.MessageEvent, recipients []string) ([]string, error) {
	// Check is recipients re valid
	var verified []string
	for _, s := range recipients {
		recipient := strings.Replace(s, "@", "", -1)
		recipient = strings.ToUpper(recipient)

		// Can't give yourself points
		if recipient == ev.User {
			return nil, errors.New("no patting yourself on the back")
		}
		// Can only thank real people. Not rubber ducks.
		recipientData, err := rtm.GetUserInfo(recipient)

		if err != nil {
			errResp := fmt.Sprintf("can't find %s to give them any %s", s, configuration.Plural)
			return nil, errors.New(errResp)
		}

		verified = append(verified, recipientData.Name)
	}
	return verified, nil
}

func main() {
	configuration := Configuration{}
	err := gonfig.GetConf("./carrots.json", &configuration)
	if err != nil {
		fmt.Printf("%s loading default config\n", err.Error())
		configuration.Emoji = "carrot"
		configuration.Plural = "carrots"
	}

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

				carrotsMatch := regexp.MustCompile(fmt.Sprintf(":%s:", configuration.Emoji))
				usersMatch := regexp.MustCompile("@([A-Za-z]+[A-Za-z0-9-_]+)")
				carrots := carrotsMatch.FindAllStringIndex(text, -1)
				recipients := usersMatch.FindAllString(text, -1)

				if ev.User != info.User.ID && len(carrots) > 0 && len(recipients) > 0 {
					sender, _ := rtm.GetUserInfo(ev.User)
					// verify recipients are valid
					verified, err := verifyRecipients(&configuration, rtm, ev, recipients)

					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(err.Error(), ev.Channel))
					} else {
						//Genuine Kudos!
						//save to db TDB
						botResp := ""
						fmt.Printf("%s sent %d %s to %d verified users\n", sender.Name, len(carrots), configuration.Plural, len(verified))
						for i := 0; i < (len(carrots) * len(verified)); i++ {
							botResp = fmt.Sprintf("%s :%s:", botResp, configuration.Emoji)
						}
						botResp = fmt.Sprintf("%s :heart:", botResp)
						rtm.SendMessage(rtm.NewOutgoingMessage(botResp, ev.Channel))
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
