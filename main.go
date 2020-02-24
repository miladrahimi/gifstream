package main

import (
	"errors"
	"gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"
)

const Token = "574147246:AAEs6jJsQrg2nFsFqgR7EajeWDZ18MhfnAs"
const DefaultUrl = "https://c5952bb4.ngrok.io/"

var servers = [][]string{
	[]string{"https://www.sex.com/gifs/", "data\\-src=\"([^\"]+)\""},
	[]string{"https://www.sex.com/gifs/hardcore/", "data\\-src=\"([^\"]+)\""},
	[]string{"http://porn-gifs.top", "data\\-highres=\"([^\"]+)\""},
	[]string{"https://www.porn.com/gifs", "src=\"(.+?\\.gif)\""},
	[]string{"http://pornopoke.com/porn-gif-xxx/", "src=\"([^\"]+?)\""},
	[]string{"http://www.porngif.org/", "src=\"(.+?\\.gif)\""},
	[]string{"http://101hotguys.com/category/gay-porn-gifs/", "src=\"(.+?\\.gif)\""},
	[]string{"https://www.eporner.com/gifs/", "src=\"(.+?\\.gif)\""},
	[]string{"http://www.pornosexgif.org/category/sex-gif/", "src=\"(.+?\\.gif)\""},
	[]string{"https://www.utporn.com/gifs", "src=\"(.+?\\.gif)\""},
	[]string{"http://zhuxian.info/reality/sexy-tori-black-porn-gif.html", "http://[^/]+?/[^\\.]+?\\.gif"},
	[]string{"http://gif-porn.net/", "src=\"(.+?\\.gif)\""},
	[]string{"http://www.gifsfor.com/", "src=\"(.+?\\.gif)\""},
	[]string{"http://www.gifsfor.com/", "src=\"(.+?\\.gif)\""},
}

var blacklistedPatterns = []string{
	"rating",
	"loading",
}

func fetchUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.New("cannot load the page")
	}

	defer resp.Body.Close()

	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("cannot parse the body")
	}

	return string(html), nil
}

func discover(htmlCache map[string]string) (string, error) {
	rand.Seed(time.Now().Unix())
	server := servers[rand.Intn(len(servers))]
	log.Print("Selected server: " + server[0])

	var html string
	var err error

	key := time.Now().Local().Format("2006-01-02") + ":" + server[0]

	if value, success := htmlCache[key]; success {
		log.Print("Load HTML from cache")
		html = value
	} else {
		html, err = fetchUrl(server[0])
		if err != nil {
			return "", err
		}

		htmlCache[key] = html
	}

	regex1, _ := regexp.Compile(server[1])
	fetchedItems := regex1.FindAllString(html, -1)

	items := make([]string, 0)

	for _, fetchedItem := range fetchedItems {
		match, _ := regexp.MatchString("http", fetchedItem)

		if match == false {
			continue
		}

		for _, blacklistedPattern := range blacklistedPatterns {
			match, _ := regexp.MatchString(blacklistedPattern, fetchedItem)

			if match {
				continue
			}
		}

		items = append(items, fetchedItem)
	}

	if len(items) == 0 {
		return "", errors.New("no gif in the selected server")
	}

	rand.Seed(time.Now().Unix())
	item := items[rand.Intn(len(items))]

	regex2, err := regexp.Compile("http[^\"]+")
	if err != nil {
		return "", err
	}

	return regex2.FindString(item), nil
}

func main() {
	bot, err := tgbotapi.NewBotAPI(Token)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	url := os.Getenv("URL")
	if url == "" {
		url = DefaultUrl
	}

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(url + bot.Token))
	if err != nil {
		log.Fatal(err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("[Telegram callback failed] %s", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8443"
	}

	go http.ListenAndServe("0.0.0.0:"+port, nil)

	chatIds := make([]int64, 0)
	htmlCache := make(map[string]string)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		isMember := false
		for _, chatId := range chatIds {
			if chatId == update.Message.Chat.ID {
				isMember = true
			}
		}

		if isMember == false {
			chatIds = append(chatIds, update.Message.Chat.ID)
		}

		gif, err := discover(htmlCache)

		if err != nil {
			log.Print(err)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
		} else {
			log.Print("Selected GIF: " + gif)

			for _, chatId := range chatIds {
				bot.Send(tgbotapi.NewDocumentShare(chatId, gif))
			}
		}
	}
}
