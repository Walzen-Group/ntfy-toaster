package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	toast "github.com/Walzen-Group/golang-toast-11"
	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
	"github.com/kyokomi/emoji"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

//go:embed assets/ntfy.ico
var iconIco []byte

type Config struct {
	Topics map[string]Topic `yaml:"topics"`
}

type Topic struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

var (
	configPath  = filepath.Join(os.Getenv("APPDATA"), "wlzntfytoaster")
	configName  = "config.yaml"
	configFile  = filepath.Join(configPath, configName)
	log         = logrus.New()
	config      *Config
	configLock  = new(sync.RWMutex)
	cancelFuncs = make(map[string]context.CancelFunc)
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	log.SetLevel(logrus.InfoLevel)
}

func loadConfig() (*Config, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		_ = os.MkdirAll(configPath, os.ModePerm)
		defaultConfig := &Config{
			Topics: map[string]Topic{
				"your_topic": {
					URL:   "your_topic_url",
					Token: "your_token (optional)",
				},
			},
		}
		data, err := yaml.Marshal(defaultConfig)
		if err != nil {
			log.Fatalf("Could not marshal default config: %v", err)
		}
		err = os.WriteFile(configFile, data, 0644)
		if err != nil {
			log.Fatalf("Could not write config file: %v", err)
		}
		log.Printf("Default config created, please configure it in %s", configFile)
		os.Exit(0)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}

	return &cfg, nil
}

func watchConfig(watchPath string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Error creating watcher: %v", err)
		return nil, err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Write) {
					cfg, err := loadConfig()
					if err != nil {
						log.Errorf("Error reloading config: %v", err)
					} else {
						configLock.Lock()
						config = cfg
						configLock.Unlock()
						syncSubscriptions()
					}
                }
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(watchPath)
	if err != nil {
		log.Errorf("Error adding file to watcher: %v", err)
		return nil, err
	}
	return watcher, nil
}

func subscribe(ctx context.Context, topic Topic, messages chan<- map[string]interface{}) {
	url := fmt.Sprintf("%s/json", topic.URL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error creating request: %v", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", topic.Token))

	log.Infof("Subscribing to %s", url)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Error subscribing to %s: %v", url, err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Error closing response body: %v", err)
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	for {
		select {
		case <-ctx.Done():
			log.Infof("Stopping subscription to %s", url)
			return
		default:
			if scanner.Scan() {
				var data map[string]interface{}
				err := json.Unmarshal(scanner.Bytes(), &data)
				if err != nil {
					log.Errorf("Error parsing JSON: %v", err)
					continue
				}
				log.Debugf("Received data: %v", data)
				messages <- data
			} else if err := scanner.Err(); err != nil {
				log.Errorf("Error reading response: %v", err)
				return
			}
		}
	}
}

func showNotification(data map[string]interface{}, topicURL string) {
	var title, message, tags string

	if t, ok := data["title"].(string); ok {
		title = t
	}

	if m, ok := data["message"].(string); ok {
		message = m
	}

	if t, ok := data["tags"].([]interface{}); ok {
		var emojis []string
		var nonEmojiTags []string
		for _, tag := range t {
			tagStr := tag.(string)
			if emojiStr := emoji.Sprint(fmt.Sprintf(":%s:", tagStr)); emojiStr != fmt.Sprintf(":%s:", tagStr) {
				emojis = append(emojis, emojiStr)
			} else {
				nonEmojiTags = append(nonEmojiTags, fmt.Sprintf("#%s", tagStr))
			}
		}
		if len(emojis) > 0 {
			title = strings.Join(emojis, " ") + title
		}
		if len(nonEmojiTags) > 0 {
			tags = strings.Join(nonEmojiTags, " ")
			message += "\n" + tags
		}
	}

	ex, err := os.Executable()
	if err != nil {
		log.Errorf("Could not get executable path: %v", err)
		return
	}
	reflectionPath := filepath.Dir(ex)
	imagePath := filepath.Join(reflectionPath, "assets", "ntfy.ico")

	toastNotification := toast.Notification{
		AppID:               "Woaster Ntfy",
		Title:               title,
		Message:             message,
		Icon:                imagePath,
		Attribution:         fmt.Sprintf("via %s", topicURL),
		ActivationType:      "protocol",
		ActivationArguments: topicURL,
	}

	if err := toastNotification.Push(); err != nil {
		log.Errorf("Error showing notification: %v", err)
	}
}

func handleMessages(messages <-chan map[string]interface{}, topicURL string) {
	for data := range messages {
		log.Infof("received message %v for topic %s", data, topicURL)
		if data["event"] == "message" {
			showNotification(data, topicURL)
		}
	}
}

func syncSubscriptions() {
	configLock.RLock()
	defer configLock.RUnlock()

	// Cancel existing subscriptions
	for _, cancel := range cancelFuncs {
		cancel()
	}
	cancelFuncs = make(map[string]context.CancelFunc)

	// Start new subscriptions
	for key, topic := range config.Topics {
		messages := make(chan map[string]interface{})
		ctx, cancel := context.WithCancel(context.Background())
		cancelFuncs[key] = cancel
		go subscribe(ctx, topic, messages)
		go handleMessages(messages, topic.URL)
	}
}

func onReady() {
	var err error
	config, err = loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	syncSubscriptions()

	systray.SetIcon(iconIco)
	systray.SetTitle("Walzen Ntfy Toast Client")

	mConfig := systray.AddMenuItem("Open Config", "Open the configuration file")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mConfig.ClickedCh:
				openExplorer(configPath)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Cleanup
	for _, cancel := range cancelFuncs {
		cancel()
	}
}

func openExplorer(path string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("explorer", path)
		_ = cmd.Run()
	} else {
		log.Errorf("Unsupported platform: %s", runtime.GOOS)
		return
	}
}

func main() {

	watcher, err := watchConfig(configFile)
	if err != nil {
		log.Fatalf("Error watching config: %v", err)
	}
	defer watcher.Close()

	systray.Run(onReady, onExit)
}
