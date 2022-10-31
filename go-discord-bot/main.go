package main

import (
	"fmt"
	"io"
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
	Token           string `yaml:"token"`
	ChannelID       string `yaml:"channel"`
	ThresholdSetAPI string `yaml:"ThresholdSetAPI"`
	EnableSetAPI    string `yaml:"EnableSetAPI"`
	DisableSetAPI   string `yaml:"DisableSetAPI"`
	RestTime        int    `yaml:"RestTime"`
	ListenAddr      string `yaml:"ListenAddr"`
}

var (
	ChannelID string
)
var cfg config

func parseConfig(file string) (config, error) {
	content, err := ioutil.ReadFile(file)

	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(content, &cfg)

	if err != nil {
		return cfg, err
	}

	return cfg, err
}

var (
	counter         int       = 6
	default_counter int       = 6
	restTime        time.Time = time.Now()
	lastTimeUpdate  time.Time = time.Now()
	logFilePath     string    = "/tmp/serveHTTP"
	cfgFilePath     string    = "/etc/config.yaml"
)

type NotifyInterface interface {
	ChannelFileSendWithMessage(config, string, io.Reader) error
	Open() error
	Close() error
}

type DiscordAdaper struct {
	adaptee *discordgo.Session
}

var notifierInterface NotifyInterface = nil

func (tee *DiscordAdaper) ChannelFileSendWithMessage(cfg config, fileName string, rdr io.Reader) error {
	str := time.Now().Format("2006-01-02 15:04:05 cnt: ") + strconv.Itoa(counter)
	_, err := tee.adaptee.ChannelFileSendWithMessage(cfg.ChannelID, str, fileName, rdr)
	return err
}
func (tee *DiscordAdaper) Open() error {
	return tee.adaptee.Open()
}

func (tee *DiscordAdaper) Close() error {
	return tee.adaptee.Close()
}

type HardwareInterface interface {
	Enable() error
	Disable() error
	setParam(string, string) error
}

type JetsonNano struct {
	addr string
}

func (j *JetsonNano) Enable() error {
	var resp *http.Response
	var err error
	resp, err = http.Get(cfg.EnableSetAPI)

	if err != nil {
		log.Println(err)
		return fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()
	return nil
}

func (j *JetsonNano) Disable() error {

	var resp *http.Response
	resp, err := http.Get(cfg.DisableSetAPI)

	if err != nil {
		log.Println(err)
		return fmt.Errorf("%v", err)
	}

	defer resp.Body.Close()
	return nil
}

func (j *JetsonNano) setParam(k string, v string) error {
	var err error
	switch k {
	case "threshold":
		var value int64
		value, err = strconv.ParseInt(v, 10, 32)
		if err != nil {
			fmt.Println(err)
			break
		}
		err = setThreshold(int(value))
		break
	default:
		break
	}

	return err
}

func setThreshold(threshold int) error {
	request_str := cfg.ThresholdSetAPI + strconv.Itoa(threshold)
	resp, err := http.Get(request_str)
	if err != nil {
		err = fmt.Errorf("%v", err)
	} else {
		defer resp.Body.Close()
	}
	return err

}
func serveHTTP(r http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	if time.Now().After(restTime) ||
		time.Now().After(lastTimeUpdate.Add(time.Duration(cfg.RestTime)*time.Minute)) {
		counter = default_counter
		restTime = time.Now().Add(time.Duration(cfg.RestTime) * time.Minute)
	} else {
		if counter > 0 {
			counter--
		}
	}
	if counter > 0 {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error when reading %v\n", err)
			http.Error(r, "error", http.StatusBadRequest)
		} else {
			rdr, err := os.Open(string(body))
			if err != nil {
				log.Println(err)
				r.WriteHeader(http.StatusInternalServerError)
			} else {
				if notifierInterface != nil {
					err = notifierInterface.ChannelFileSendWithMessage(cfg, string(body), rdr)

					if err != nil {
						log.Println(err)
						r.WriteHeader(http.StatusInternalServerError)
					}
				} else {
					log.Println("notifierInterface is nil")
					r.WriteHeader(http.StatusInternalServerError)
				}
				if err == nil {
					r.Write([]byte("ok"))
				}
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

func initDiscord() NotifyInterface {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalf("error creating Discord session %v ", err)
		return nil
	}

	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	dgSession := new(DiscordAdaper)
	dgSession.adaptee = dg
	notifierInterface = dgSession

	for {
		err = notifierInterface.Open()
		if err != nil {
			log.Println("error opening connection,", err)
		} else {
			break
		}
		time.Sleep(5 * time.Second)
	}

	dg.ChannelMessageSend(ChannelID, "Bot started")
	return notifierInterface
}

func initLog() *os.File {

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
		return nil
	}
	log.SetOutput(logFile)
	return logFile
}

var (
	jetson HardwareInterface
)

func initHardware() HardwareInterface {
	jetson = new(JetsonNano)
	return jetson
}
func main() {
	logFile := initLog()
	initHardware()
	defer logFile.Close()
	cfg, err := parseConfig(cfgFilePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(cfg)

	ChannelID = cfg.ChannelID
	go runServer()
	dg := initDiscord()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	var res string = "ok"
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	if jetson == nil {
		log.Println("jetson is null")
		return
	}

	switch m.Content {
	case "!enable":
		err := jetson.Enable()
		counter = default_counter
		restTime = time.Now()
		if err != nil {
			log.Println(err)
			res = err.Error()
		}
		s.ChannelMessageSend(m.ChannelID, res)

	case "!disable":
		err := jetson.Disable()
		if err != nil {
			log.Println(err)
			res = err.Error()
		}
		s.ChannelMessageSend(m.ChannelID, res)

	case "!reb":
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)

	default:
		if strings.Contains(m.Content, "!threshold") {
			substr := strings.Split(m.Content, " ")
			var err error
			if len(substr) != 2 {
				res = "invalid command"
			} else {
				err = jetson.setParam("threshold", substr[1])
				if err != nil {
					res = err.Error()
				}
			}
			s.ChannelMessageSend(m.ChannelID, res)
		}
	}

}
