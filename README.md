# Carrots

Golang KudosBot for Slack

## Installation

* Setup the Bots app for you slack workspace: <https://app.slack.com/apps/A0F7YS25R-bots?next_id=0>, save the token somewhere secret
* Create the MySQL db, I have included the commands I used
* Update the carrots.json file
* Build the app from the repo working directory

    ```sh
    go get
    go build 
    ```

* Run the app

    ```sh
    SLACKTOKEN=[SLACK-TOKEN] MYSQLPASS=[DB-PASSWORD] ./carrots
    ```

* Invite the bot to your channel

## Configuration

* All config options are in `carrots.json`. 
* "Limits", is the number of emoji a user can send in a calendar month, `-1` removes the limit
* "PerUserLimit" is the number of emoji a user can send per message, `-1` removes the limit 

## Usage

### Commands

> `@kudos help` Get help, only visible to the message sender  
> `@kudos ladder` Print the monthly leaderboard  
> `@kudos me` Print your personal monthly stats, only visible to the message sender  

### Sending Kudos

```
@<person> <emoji>
or
<emoji> @<person1> @<person2>
or
Some <emoji> <emoji> are due to @<person> for being awesome
```
