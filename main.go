package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var token string
var buffer = make([][]byte, 0)
var soundsBuffers = make(map[string][][]byte)

var stackTime = 49        // ingame time to stack
var stackDelay = 60  // interval between stack
var bountyRunesTime = 173 //180
var bountyRunesDelay = 180
var riverRunesTime = 110  // 120
var gameTime = 0
var gameDone = make(chan bool)

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
	// Iterate through the
	for key, value := range soundsBuffers {
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

	ticker := time.NewTicker(1 * time.Second)

	if m.Content == "!start" {
		_, g, _ := getChannelAndGuild(s, m)

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				// Join the provided voice channel.
				vc, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
				if err != nil {
					fmt.Println("Error joining channel: ", err)
					return
				}

				playSpecificSound(vc, soundsBuffers["diego.dca"])
				go startGame(ticker, &gameTime, vc)
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	}

	if m.Content == "!time" {
		// message := strconv.Itoa(gameTime) + " Seconds"
		message2 := secondsToMinutes(gameTime)

		_, err := s.ChannelMessageSend(m.ChannelID, message2)
		if err != nil {
			fmt.Println(err)
		}
	}

	if m.Content == "!rita" {
		_, g, _ := getChannelAndGuild(s, m)

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				// Join the provided voice channel.
				vc, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
				if err != nil {
					fmt.Println("Error joining channel: ", err)
					return
				}

				go playSpecificSound(vc, soundsBuffers["rita.dca"])
			}
		}
	}

	if m.Content == "!quit" {
		_, g, _ := getChannelAndGuild(s, m)

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				// Join the provided voice channel.
				vc, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
				if err != nil {
					fmt.Println("Error joining channel: ", err)
					return
				}

				vc.Disconnect()
				ticker.Stop()
				gameDone <- true
				gameTime = 0
        fmt.Println("Game ended")
			}
		}
	}

	// Add time(seconds) to the game time
	if strings.HasPrefix(m.Content, "!add") {
		words := strings.Fields(m.Content)
		secondsToAdd := words[1]
		i, _ := strconv.Atoi(secondsToAdd)
		gameTime += i
	}

	// Subtract time (seconds) from the game time
	if strings.HasPrefix(m.Content, "!remove") {
		words := strings.Fields(m.Content)
		secondsToAdd := words[1]
		i, _ := strconv.Atoi(secondsToAdd)
		gameTime -= i
	}
}

func playSpecificSound(vc *discordgo.VoiceConnection, audioBuffers [][]byte) {
	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	// Send the buffer data.
	for _, buff := range audioBuffers {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)
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

// Routine to be called when the game started
func startGame(ticker *time.Ticker, gameTime *int, vc *discordgo.VoiceConnection) {
  fmt.Println("Game started")
	for {
		select {
		case <-gameDone:
			return
		case <-ticker.C:
			*gameTime += 1
			// fmt.Println("Game time", *gameTime)

			if (*gameTime-stackTime)%stackDelay == 0 {
      // TODO: This will block the tick execution, should own it own thread
				go playSpecificSound(vc, soundsBuffers["stack.dca"])
			}

			if (*gameTime-bountyRunesTime)%bountyRunesDelay == 0 {
				go playSpecificSound(vc, soundsBuffers["runa.dca"])
			}

		}
	}
}

// Get the channel and guild that the message is coming from
func getChannelAndGuild(s *discordgo.Session, m *discordgo.MessageCreate) (c *discordgo.Channel, g *discordgo.Guild, err error) {
	// Find the channel that the message came from.
	c, err = s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Find the guild for that channel.
	g, err = s.State.Guild(c.GuildID)
	if err != nil {
		// Could not find guild.
		return
	}

	return
}

func secondsToMinutes(inSeconds int) string {
	minutes := inSeconds / 60
	seconds := inSeconds % 60
  str2 := fmt.Sprintf("%02d:%02d\n", minutes, seconds)
	return str2
}
