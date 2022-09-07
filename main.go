package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var token string

func init() {
	// This will get the value passed to the program on the flag -t to the token variable
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	if token == "" {
		fmt.Printf("You need to pass the token, please run ./belphegor -t <token value>")
		return
	}

	discord, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println("Error creating the discord session")
		return
	}

	// pass a event and a function to handle the event https://discord.com/developers/docs/topics/gateway#event-names
	discord.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	// discord.Identify.Intents = discordgo.IntentsGuildMessages
	discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
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
	discord.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		fmt.Println("Nao eh pra entrar")
		return
	}

	randomQuoteApiBaseUrl := "https://api.quotable.io"
	fmt.Println("Content value" + m.Content)

	if m.Content == "!diegoBaitola" {
		//Call the quote API and retrieve a random quote
		response, err := http.Get(randomQuoteApiBaseUrl + "/random/")

		if err != nil {
			fmt.Println(err)
		}

		defer response.Body.Close()

		if response.StatusCode == 200 {
			_, err := s.ChannelMessageSend(m.ChannelID, "testing")
			// _, err = s.ChannelFileSend(m.ChannelID, "random-gopher.png", response.Body)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Error: Can't get random quote! :-(")
		}
	}
}
