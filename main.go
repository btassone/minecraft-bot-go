package main

import (
	"flag"
	"log"
	"minecraft-discord-bot/config"
	"minecraft-discord-bot/discord"
	"minecraft-discord-bot/provider"
)

func init() {
	flag.Parse()

	_, err := provider.MakeEc2Api()
	if err != nil {
		log.Fatalln(err)
	}

	err = provider.Api.Setup(provider.Ec2CredentialsInput{
		Id:      *config.Id,
		Secret:  *config.Secret,
		Session: "",
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	defer discord.Session.Close()

	discord.SetupSession()
	discord.SignalWait()
	discord.Shutdown()
}
