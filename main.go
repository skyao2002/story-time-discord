package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var textSynthURL = "https://api.textsynth.com/v1/engines/gptj_6B/completions"

var (
	BearerToken string
	Bearer      string
	BotToken    string
	GuildID     string
)

func init() {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	BotToken = os.Getenv("BOT_TOKEN")
	if BotToken == "" {
		log.Fatal("$BOT_TOKEN must be set")
	}

	BearerToken = os.Getenv("BEARER_TOKEN")
	if BearerToken == "" {
		log.Fatal("$BEARER_TOKEN must be set")
	}
	flag.StringVar(&GuildID, "guild", "", "Guild ID to test in")
	flag.Parse()
	Bearer = "Bearer " + BearerToken
}

var s *discordgo.Session

func init() {
	var err error
	s, err = discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

type Story struct {
	Text string `json: "text"`
}

func callTextSynth(prompt string, maxTokens int) string {
	contentJSON := map[string]interface{}{
		"prompt":     prompt,
		"max_tokens": maxTokens,
	}
	content, _ := json.Marshal(contentJSON)

	req, err := http.NewRequest("POST", textSynthURL, bytes.NewBuffer(content))
	if err != nil {
		log.Println(err)
		return ""
	}

	// add authorization header to the req
	req.Header.Add("Authorization", Bearer)
	req.Header.Add("Content-Type", "application/json")

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		var story Story
		err = json.Unmarshal(body, &story)
		return story.Text
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("Error: Non 200 status code received :-(")
		log.Println(string(body))
		return ""
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "story",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "I generate a story based on your given prompt",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "Enter the words you want your story to start with.",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "words",
					Description: "Number of words to generate. Default is 100",
					Required:    false,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"story": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, we'll discuss them in "responses" part
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "*Generating story (this may take a couple seconds)...*",
				},
			})

			err := userAccess(i.Interaction.Member.User.ID, i.Interaction.Member.User.Username)
			if err != nil {
				var errorMsg string
				if _, ok := err.(*tooManyRequestsError); ok {
					errorMsg = err.Error()
				} else {
					errorMsg = "Unknown Error with Database occurred. "
				}
				s.InteractionResponseEdit(s.State.User.ID, i.Interaction, &discordgo.WebhookEdit{
					Content: errorMsg,
				})

				return
			}

			prompt := i.ApplicationCommandData().Options[0].StringValue()
			numWords := 100

			if len(i.ApplicationCommandData().Options) >= 2 {
				numWords = int(i.ApplicationCommandData().Options[1].IntValue())
			}

			// log.Println("story called with numWords " + strconv.Itoa(numWords))

			story := callTextSynth(prompt, numWords)
			// log.Println("Successfully called api, which returned " + story)
			var output string
			if story != "" {
				output = "**" + prompt + "**" + story
			} else {
				output = "The model is currently down. Try again in a couple of hours."
			}

			s.InteractionResponseEdit(s.State.User.ID, i.Interaction, &discordgo.WebhookEdit{
				Content: output,
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
	}

	defer s.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Gracefully shutdowning")
}

// // This function will be called (due to AddHandler above) every time a new
// // message is created on any channel that the authenticated bot has access to.
// func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

//     // Ignore all messages created by the bot itself
//     // This isn't required in this specific example but it's a good practice.
//     if m.Author.ID == s.State.User.ID || m.Content[0] != Prefix || len(m.Content) <= 7 {
//         return
//     }

//     if m.Content[:7] == "!story " {
// 		prompt := m.Content[7:]
// 		content := []byte(`{"prompt": "` + prompt + `"}`)

// 		req, err := http.NewRequest("POST", TextSynthURL, bytes.NewBuffer(content))
// 		if err != nil {
//             fmt.Println(err)
//         }

// 		// add authorization header to the req
// 		req.Header.Add("Authorization", Bearer)
// 		req.Header.Add("Content-Type", "application/json")

// 		// Send req using http Client
// 		client := &http.Client{}
// 		resp, err := client.Do(req)
//         if err != nil {
// 			panic(err)
// 		}
// 		defer resp.Body.Close()

//         if resp.StatusCode == 200 {
// 			body, _ := ioutil.ReadAll(resp.Body)

// 			var story Story
// 			err = json.Unmarshal(body, &story)
// 			msg := "**" + prompt + "**" + story.Text
//             // Send a text message to the channel
//             _, err = s.ChannelMessageSend(m.ChannelID, msg)
//             if err != nil {
//                 fmt.Println(err)
//             }
//         } else {
//             fmt.Println("Error: Non 200 status code received :-(")
//         }
//     }
// }
