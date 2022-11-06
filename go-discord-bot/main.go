package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	UploadConfigAPI string `yaml:"UploadConfigAPI"`
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
	counter         int       = 3
	default_counter int       = 3
	restTime        time.Time = time.Now()
	lastTimeUpdate  time.Time = time.Now()
	logFilePath     string    = "/tmp/serveHTTP"
	cfgFilePath     string    = "/etc/config.yaml"
)

type NotifyInterface interface {
	ChannelFileSendWithMessage(config, string, io.Reader) error
	ChannelMessageSend(config, string) error
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

func (tee *DiscordAdaper) ChannelMessageSend(cfg config, content string) error {
	_, err := tee.adaptee.ChannelMessageSend(cfg.ChannelID, content)
	return err
}

func (tee *DiscordAdaper) Open() error {
	return tee.adaptee.Open()
}

func (tee *DiscordAdaper) Close() error {
	return tee.adaptee.Close()
}

type HardwareInterface interface {
	UploadConfig(string) error
}

type JetsonNano struct {
	addr string
}

func (j *JetsonNano) UploadConfig(cf string) error {
	var resp *http.Response
	var err error
	resp, err = http.Post(cfg.UploadConfigAPI, "application/json", strings.NewReader(cf))

	if err != nil {
		log.Println(err)
		return fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()
	return nil
}

type JsonData struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

func ParseJsonData(data string) (JsonData, error) {
	var jdata JsonData
	rdr := strings.NewReader(data)
	de := json.NewDecoder(rdr)
	err := de.Decode(&jdata)

	if err != nil {
		return JsonData{}, err
	}
	return jdata, nil
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
			data, err := ParseJsonData(string(body))
			if err != nil {
				r.WriteHeader(http.StatusInternalServerError)
			} else {
				if data.Key == "img_path" {
					rdr, err := os.Open(data.Content)
					if err != nil {
						log.Println(err)
						r.WriteHeader(http.StatusInternalServerError)
					} else {
						if notifierInterface != nil {
							err = notifierInterface.ChannelFileSendWithMessage(cfg, "img.jpg", rdr)

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
				} else {
					if notifierInterface != nil {
						notifierInterface.ChannelMessageSend(cfg, data.Content)
					}
				}
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
	localDevice HardwareInterface
)

func initHardware() HardwareInterface {
	localDevice = new(JetsonNano)
	return localDevice
}

func main() {
	//logFile := initLog()
	initHardware()
	//defer logFile.Close()
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

type Detectionconfig struct {
	BoardName             string
	CfgFile               string
	CorrectRate           float32
	Delay4CAP             int16
	NameFile              string
	NotifyAPI             string
	Port                  int16
	QUEUE_ENTRY_LIMIT_MIN int16
	Src                   string
	TIME_FORCUS           int16
	TIME_SKIP             int16
	Threshold             float32
	WeightFile            string
	Status                bool
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	var res string = "ok"
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case "!reb":
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	case "!status":
		s.ChannelMessageSend(m.ChannelID, readStatus())
	default:

		if localDevice == nil {
			log.Println("localDevice is null")
			return
		}
		rdr := strings.NewReader(m.Content)
		de := json.NewDecoder(rdr)
		var ret Detectionconfig
		err := de.Decode(&ret)
		if err != nil {
			res = err.Error()
		} else {
			log.Println(ret)
			by, err := json.Marshal(ret)
			if testDetectionConfig(ret) == false || err != nil {
				res = "cfg error"
			} else if err := localDevice.UploadConfig(string(by)); err != nil {
				res = err.Error()
			}
		}
		s.ChannelMessageSend(m.ChannelID, res)
	}
}

func testDetectionConfig(cf Detectionconfig) bool {
	if cf.BoardName == "" || cf.CfgFile == "" || cf.NameFile == "" || cf.NotifyAPI == "" || cf.Src == "" || cf.WeightFile == "" {
		return false
	}

	if cf.CorrectRate < 0.01 || cf.Delay4CAP == 0 || cf.Port == 0 || cf.QUEUE_ENTRY_LIMIT_MIN == 0 || cf.TIME_FORCUS == 0 || cf.TIME_SKIP == 0 || cf.Threshold < 0.01 {
		return false
	}
	return true
}

func readStatus() string {
	l_cmd := exec.Command("free", "-m")
	stdout, err := l_cmd.StdoutPipe()
	if err != nil {
		return err.Error()
	}

	if err := l_cmd.Start(); err != nil {
		return err.Error()
	}

	rdr := bufio.NewReader(stdout)
	buf := ""
	for {
		line, _, err := rdr.ReadLine()
		if err != nil {
			break
		}
		buf += string(line)
		buf += "\n"
	}

	return buf
}
