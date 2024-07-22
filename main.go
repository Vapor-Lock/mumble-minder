package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/xlab/treeprint"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
)

func main() {
	BotToken := os.Getenv("MUMBLE_MINDER_BOT_TOKEN")
	TargetChannel := os.Getenv("MUMBLE_MINDER_BOT_CHANNEL")
	mumbleAddress := os.Getenv("MUMBLE_MINDER_MUMBLE_HOST")

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

	client, err := gumble.DialWithDialer(new(net.Dialer), mumbleAddress, config, tlsConfig)
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

	message := getOnline(client, config.Username)

	m, err := discord.ChannelMessageSend(TargetChannel, message)
	if err != nil {
		panic(err)
	}

	go func(discord *discordgo.Session) {
		for range time.Tick(time.Minute) {
			message := getOnline(client, config.Username)
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

func getOnline(client *gumble.Client, botUser string) string {
	rootChan := client.Channels.Find()
	tree := treeprint.New()
	treeRoot := tree.AddBranch(rootChan.Name)
	addChildren(treeRoot, rootChan, botUser)
	//[TODO] implement a completed chan to avoid the sleeps
	time.Sleep(5 * time.Second)
	return fmt.Sprintf("```%v\n````Last Update:` <t:%v:R>\n", tree.String(), time.Now().Unix())
}

func addChildren(node treeprint.Tree, channel *gumble.Channel, botUser string) {
	for _, channel := range channel.Children {
		child := node.AddBranch(channel.Name)
		for _, user := range channel.Users {
			if !strings.EqualFold(user.Name, botUser) {
				child.AddNode(user.Name)
			}
		}
		go addChildren(child, channel, botUser)
	}
}
