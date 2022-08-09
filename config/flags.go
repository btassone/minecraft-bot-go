package config

import "flag"

var (
	Id        = flag.String("id", "", "The id of the aws user")
	Secret    = flag.String("secret", "", "The secret of the aws user")
	AppId     = flag.String("app", "", "The app id of the discord bot")
	GuildId   = flag.String("guild", "", "The guild id of the discord bot")
	ChannelId = flag.String("channel", "", "The channel id for the bot responses")
	RoleId    = flag.String("role", "", "The role id used for commands")
	Token     = flag.String("token", "", "The token of the bot user")
)

func init() {
	flag.Parse()
}
