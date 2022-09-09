package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var token string
var buffer = make([][]byte, 0)
var soundsBuffers = make(map[string][][]byte)

func init() {
	// This will get the value passed to the program on the flag -t to the token variable
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func createSoundsBufferMap() {
	items, _ := ioutil.ReadDir("./sounds/")
	fmt.Printf("Found %d sounds\n", len(items))

	for _, soundItem := range items {
		soundsBuffers[soundItem.Name()] = make([][]byte, 0)
	}
}

func main() {
	if token == "" {
		fmt.Printf("You need to pass the token, please run ./belphegor -t <token value>")
		return
	}

	createSoundsBufferMap()
	fmt.Println("soundsBuffers", soundsBuffers)
	// Iterate through the
	for key, value := range soundsBuffers {
		fmt.Println("Loading sound key value", key, value)
		err := loadSound(value, key, soundsBuffers)
		if err != nil {
			fmt.Println("Error loading sound: ", err)
			fmt.Println("Please copy a file.dca to this directory.")
			return
		}
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

// loadSound attempts to load an encoded sound file from disk.
func loadSound(sBuffer [][]byte, sName string, soundsBuffers map[string][][]byte) error {
	file, err := os.Open("./sounds/" + sName)

	if err != nil {
		fmt.Println("Something went worng opening audio file:", err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Append encoded pcm data to the buffer.
		sBuffer = append(sBuffer, InBuf)
    soundsBuffers[sName] = sBuffer
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
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

	if m.Content == "!aipapai" {
		fmt.Println("Checking the channel", m.ChannelID)
		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		fmt.Println("Checking guild", c.GuildID)
		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID, soundsBuffers["diego.dca"])
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string, sBuffer [][]byte) (err error) {

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	// Send the buffer data.
	for _, buff := range sBuffer {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
