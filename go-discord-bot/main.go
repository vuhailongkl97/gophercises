package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

var (
	dg        *discordgo.Session
	ChannelID string
)

func parseConfig(file string) (config, error) {
	content, err := ioutil.ReadFile(file)
	var cfg config

	if err != nil {
		log.Fatal(err)
		return cfg, err
	}
	err = yaml.Unmarshal(content, &cfg)

	if err != nil {
		log.Fatal(err)
		return cfg, err
	}

	return cfg, nil
}

func serveHTTP(r http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		log.Printf("Error when reading %v\n", err)
		http.Error(r, "error", http.StatusBadRequest)
	} else {
		r.Write([]byte("ok"))
		fmt.Printf("body is [%v]\n", string(body))
		rdr, err := os.Open(string(body))
		if err != nil {
			fmt.Println(err)
		} else {
			if dg != nil {
				_, err := dg.ChannelFileSendWithMessage(ChannelID, time.Now().String(), "wallpapers.png", rdr)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("dg is nil")
			}

		}
	}
}
func runServer() {
	http.HandleFunc("/updated", serveHTTP)
	err := http.ListenAndServe(":1234", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	// get Token and Channel ID in Yaml file
	cfg, err := parseConfig("config.yaml")

	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(cfg)

	ChannelID = cfg.ChannelID
	go runServer()

	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + cfg.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	for {
		err = dg.Open()
		if err != nil {
			fmt.Println("error opening connection,", err)
		} else {
			break
		}
		time.Sleep(5 * time.Second)
	}
	//rdr, err := os.Open("wallpapers.jpg")

	dg.ChannelMessageSend(ChannelID, "Bot started")
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

	fmt.Println(m.ChannelID)
	switch m.Content {
	case "!enable":
		fmt.Println("enable iot")
	case "!disable":
		fmt.Println("disable iot")
	}

}
