package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-yaml/yaml"
)

type config struct {
	Token     string `yaml:"token"`
	ChannelID string `yaml:"channel"`
}

func main() {

	// get Token and Channel ID in Yaml file
	content, err := ioutil.ReadFile("config.yaml")

	if err != nil {
		log.Fatal(err)
	}
	var cfg config
	err = yaml.Unmarshal(content, &cfg)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg)

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!random" {
		rdr, err := os.Open("wallpapers.jpg")
		if err != nil {
			fmt.Println(err)
		} else {
			_, err := s.ChannelFileSendWithMessage(m.ChannelID, time.Now().String(), "wallpapers.png", rdr)
			fmt.Printf("channelID %v\n", m.ChannelID)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

}
