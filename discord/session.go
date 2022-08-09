package discord

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"minecraft-discord-bot/config"
	"minecraft-discord-bot/provider"
	"os"
	"os/signal"
	"syscall"
)

var (
	Session                *discordgo.Session
	Instances              []provider.MinecraftInstanceOutput
	SelectedInstanceId     string
	SelectedInstanceOutput provider.MinecraftInstanceOutput
	RegisteredCommands     = map[string][]*discordgo.ApplicationCommand{
		"app":    {},
		"global": {},
	}
)

func init() {
	var err error
	Session, err = discordgo.New(fmt.Sprintf("Bot %s", *config.Token))
	if err != nil {
		log.Fatalln(err)
	}

	err = Session.Open()
	if err != nil {
		log.Fatalln(err)
	}
}

func SetupSession() {
	minValue := 1
	appCmds, err := Session.ApplicationCommands(*config.AppId, *config.GuildId)
	if err != nil {
		log.Fatalln(err)
	}

	for _, appCmd := range appCmds {
		RegisteredCommands["app"] = append(RegisteredCommands["app"], appCmd)
	}

	globalCmds, err := Session.ApplicationCommands(*config.AppId, "")
	if err != nil {
		log.Fatalln(err)
	}

	for _, globalCmd := range globalCmds {
		RegisteredCommands["global"] = append(RegisteredCommands["global"], globalCmd)
	}

	Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		var servers []discordgo.SelectMenuOption
		Instances = provider.Api.GetInstances()

		for _, instance := range Instances {
			servers = append(servers, discordgo.SelectMenuOption{
				Label:       instance.Name,
				Value:       instance.Id,
				Description: string(instance.Status),
			})
		}

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			var canExecute bool
			if i.Member == nil {
				return
			}

			for _, role := range i.Member.Roles {
				if role == *config.RoleId {
					canExecute = true
				}
			}

			if i.ChannelID == *config.ChannelId && canExecute {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Please select a server to change its state",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									discordgo.SelectMenu{
										CustomID:    "select-minecraft-server",
										Placeholder: "Select a server to change it's state",
										MinValues:   &minValue,
										MaxValues:   1,
										Options:     servers,
										Disabled:    false,
									},
								},
							},
						},
						Flags: uint64(discordgo.MessageFlagsEphemeral),
					},
				})
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Unable to run command in this channel",
						Flags:   uint64(discordgo.MessageFlagsEphemeral),
					},
				})
				if err != nil {
					log.Fatalln(err)
				}
			}
		case discordgo.InteractionMessageComponent:
			componentMessageId := i.MessageComponentData().CustomID

			switch componentMessageId {
			case "select-minecraft-server":
				SelectedInstanceId = i.MessageComponentData().Values[0]
				for _, instance := range Instances {
					if instance.Id == SelectedInstanceId {
						SelectedInstanceOutput = instance
					}
				}

				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Start or stop the server %s?", SelectedInstanceOutput.Name),
						Flags:   uint64(discordgo.MessageFlagsEphemeral),
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									discordgo.Button{
										Label:    "Start Server",
										Style:    discordgo.PrimaryButton,
										Disabled: SelectedInstanceOutput.Status != "stopped",
										CustomID: "start-server",
									},
									discordgo.Button{
										Label:    "Stop Server",
										Style:    discordgo.PrimaryButton,
										Disabled: SelectedInstanceOutput.Status != "running",
										CustomID: "stop-server",
									},
								},
							},
						},
					},
				})
				if err != nil {
					log.Fatalln(err)
				}
			case "start-server":
				if provider.Api.StartInstance(SelectedInstanceOutput.Id) {
					err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Started server %s", SelectedInstanceOutput.Name),
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
					if err != nil {
						log.Fatalln(err)
					}
				} else {
					err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("There was a problem starting server %s", SelectedInstanceOutput.Name),
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
					if err != nil {
						log.Fatalln(err)
					}
				}
			case "stop-server":
				if provider.Api.StopInstance(SelectedInstanceOutput.Id) {
					err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Stopped server %s", SelectedInstanceOutput.Name),
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
					if err != nil {
						log.Fatalln(err)
					}
				} else {
					err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("There was a problem stopping server %s", SelectedInstanceOutput.Name),
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
					if err != nil {
						log.Fatalln(err)
					}
				}
			}
		}
	})

	_, err = Session.ApplicationCommandCreate(*config.AppId, *config.GuildId, &discordgo.ApplicationCommand{
		Name:        "server",
		Description: "Contains commands relative to managing a minecraft server",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "state",
				Description: "Set the state of the server",
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
}

// RemoveAppCommands remove the specified app commands
func RemoveAppCommands() error {
	log.Println(fmt.Sprintf("removing %d commands from app id: %s and guild id: %s", len(RegisteredCommands["app"]), *config.AppId, *config.GuildId))
	for _, v := range RegisteredCommands["app"] {
		err := Session.ApplicationCommandDelete(*config.AppId, *config.GuildId, v.ID)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot delete '%v' command: %v", v.Name, err))
		}
	}

	return nil
}

// RemoveGlobalCommands remove the specified global commands
func RemoveGlobalCommands() error {
	log.Println(fmt.Sprintf("removing %d of global commands from app id: %s", len(RegisteredCommands["global"]), *config.AppId))
	for _, v := range RegisteredCommands["global"] {
		err := Session.ApplicationCommandDelete(*config.AppId, "", v.ID)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot delete '%v' global command: %v", v.Name, err))
		}
	}

	return nil
}

func SignalWait() {
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func Shutdown() {
	err := RemoveAppCommands()
	if err != nil {
		log.Fatalln(err)
	}

	err = RemoveGlobalCommands()
	if err != nil {
		log.Fatalln(err)
	}
}
