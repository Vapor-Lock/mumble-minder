package main

import (
	"crypto/tls"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
	"net"
	"os"
	"strings"
	"time"
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

	go func(discord *discordgo.Session) {
		for _ = range time.Tick(time.Minute) {
			var userList []string
			for _, v := range client.Users {
				userList = append(userList, v.Name)
			}

			message := strings.Join(userList, "\n")
			message = fmt.Sprintf("```%s```", message)

			discord.ChannelMessageSend(TargetChannel, message)
		}
	}(discord)

	for v := range keepAlive {
		if v {
			os.Exit(0)
		}
	}
}
