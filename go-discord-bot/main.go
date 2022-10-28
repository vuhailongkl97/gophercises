package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

var (
	counter         int       = 6
	default_counter int       = 6
	restTime        time.Time = time.Now()
	lastTimeUpdate  time.Time = time.Now()
	logFilePath     string    = "/tmp/serveHTTP"
)

func serveHTTP(r http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	if time.Now().After(restTime) {
		counter--
		if counter == 0 {
			counter = default_counter
			restTime = time.Now().Add(20 * time.Minute)
		}
		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			log.Printf("Error when reading %v\n", err)
			http.Error(r, "error", http.StatusBadRequest)
		} else {
			rdr, err := os.Open(string(body))
			if err != nil {
				log.Println(err)
			} else {
				if dg != nil {
					_, err := dg.ChannelFileSendWithMessage(ChannelID, time.Now().String(), string(body), rdr)
					if err != nil {
						log.Println(err)
					}
				} else {
					log.Println("dg is nil")
				}
				r.Write([]byte("ok"))
			}
			if counter < default_counter && lastTimeUpdate.Add(2*time.Second).After(time.Now()) {
				counter++
			}
			lastTimeUpdate = time.Now()
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

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	//	log.SetFlags(log.Lshortfile | log.LstdFlags)
	// get Token and Channel ID in Yaml file
	cfg, err := parseConfig("config.yaml")

	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println(cfg)

	ChannelID = cfg.ChannelID
	go runServer()

	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Println("error creating Discord session,", err)
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
			log.Println("error opening connection,", err)
		} else {
			break
		}
		time.Sleep(5 * time.Second)
	}
	//rdr, err := os.Open("wallpapers.jpg")

	dg.ChannelMessageSend(ChannelID, "Bot started")
	// Wait here until CTRL-C or other term signal is received.
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

	switch m.Content {
	case "!enable":
		res, err := setIOT(false)
		if err != nil {
			log.Println(err)
			res = err.Error()
		}
		s.ChannelMessageSend(m.ChannelID, res)

	case "!disable":
		res, err := setIOT(true)
		if err != nil {
			log.Println(err)
			res = err.Error()
		}
		s.ChannelMessageSend(m.ChannelID, res)
	default:

		fileStat, _ := os.Stat(logFilePath)
		log.Printf("curr log file size is %v", fileStat.Size())
		if strings.Contains(m.Content, "!threshold") {
			substr := strings.Split(m.Content, " ")
			var res string
			var err error
			if len(substr) != 2 {
				res = "invalid command"
			} else {
				threshold, _ := strconv.ParseInt(substr[1], 10, 32)
				res, err = setIOTThreshold(int(threshold))
				if err != nil {
					log.Println(err)
					res = err.Error()
				}
			}
			s.ChannelMessageSend(m.ChannelID, res)
		}
	}

}

func setIOTThreshold(threshold int) (string, error) {
	if threshold < 0 || threshold > 100 {
		return "overflow or underflow", nil
	}
	request_str := "http://localhost:18080/threshold/" + strconv.Itoa(threshold)
	resp, err := http.Get(request_str)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	return string(data), nil

}
func setIOT(disable bool) (string, error) {
	var resp *http.Response
	var err error
	if disable == true {
		resp, err = http.Get("http://localhost:18080/disable/1")
	} else {
		resp, err = http.Get("http://localhost:18080/disable/0")
	}

	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	return string(data), nil
}
