package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
)

func main() {
	BotToken := os.Getenv("MUMBLE_MINDER_BOT_TOKEN")
	TargetChannel := os.Getenv("MUMBLE_MINDER_BOT_CHANNEL")

	keepAlive := make(chan bool)
	config := gumble.NewConfig()
	config.Username = os.Getenv("MUMBLE_MINDER_USER")
	config.Password = os.Getenv("MUMBLE_MINDER_PW")
	config.Attach(gumbleutil.AutoBitrate)
	config.Attach(gumbleutil.Listener{
		Disconnect: func(e *gumble.DisconnectEvent) {
			keepAlive <- true
		},
	})

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	client, err := gumble.DialWithDialer(new(net.Dialer), "mumble.vaporlock.space:64738", config, tlsConfig)
	if err != nil {
		panic(err)
	}

	defer client.Disconnect()

	discord, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		panic(err)
	}

	discord.Open()
	defer discord.Close()

	err = emptyChannel(discord, TargetChannel)
	if err != nil {
		panic(err)
	}

	userList := getOnline(client, config.Username)
	message := fmtMessage(userList)
	m, err := discord.ChannelMessageSend(TargetChannel, message)
	if err != nil {
		panic(err)
	}

	go func(discord *discordgo.Session) {
		for range time.Tick(time.Minute) {
			userList := getOnline(client, config.Username)
			message := fmtMessage(userList)
			discord.ChannelMessageEdit(TargetChannel, m.ID, message)
		}
	}(discord)

	for v := range keepAlive {
		if v {
			os.Exit(0)
		}
	}
}

func emptyChannel(discord *discordgo.Session, targetChannel string) error {
	messages, err := discord.ChannelMessages(targetChannel, 100, "", "", "")
	if err != nil {
		return err
	}
	if len(messages) > 0 {
		var mIDs []string
		for _, v := range messages {
			mIDs = append(mIDs, v.ID)
		}
		err = discord.ChannelMessagesBulkDelete(targetChannel, mIDs)
		if err != nil {
			return err
		}
	}
	return nil
}

func getOnline(mumble *gumble.Client, botUser string) []string {
	var userList []string
	for _, v := range mumble.Users {
		if !strings.EqualFold(v.Name, botUser) {
			userList = append(userList, v.Name)
		}
	}
	return userList
}

func fmtMessage(userList []string) string {
	message := strings.Join(userList, "\n")
	message = fmt.Sprintf("```%s```", message)
	return message
}
