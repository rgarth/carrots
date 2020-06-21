# Carrots

Golang KudosBot for Slack

## Installation

* Setup the Bots app for you slack workspace: <https://app.slack.com/apps/A0F7YS25R-bots?next_id=0>, save the token somewhere secret
* Create the MySQL db, I have included the commands I used
* Update the carrots.json file
* Build the app

    ```
    go get
    go build
    ```

* Run the app

    ```
    SLACKTOKEN=xoxb-... MYSQLPASS=... ./carrots
    ```

* Invite the bot to your channel

## Configuration

All config options are in `carrots.json`. "Limits", is the number of emoji a user can send in a calendar month, `-1` removes the limit

## Usage

Once your bot has joined a channel, `@botnam help`
