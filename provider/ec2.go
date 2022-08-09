package provider

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"strconv"
)

const (
	MinecraftTagKey   = "Type"
	MinecraftTagValue = "minecraft"
)

var (
	Api Ec2Api
)

type Ec2Api struct {
	Client              *ec2.Client
	CredentialsProvider credentials.StaticCredentialsProvider
	Config              aws.Config
	Filters             map[string]types.Filter
	Context             context.Context
}

type Ec2CredentialsInput struct {
	Id      string
	Secret  string
	Session string
}

type MinecraftInstanceOutput struct {
	Name   string                  `json:"name"`
	Id     string                  `json:"id"`
	Ip     string                  `json:"ip"`
	Port   string                  `json:"port"`
	Status types.InstanceStateName `json:"status"`
}

// MakeEc2Api make and assign the Ec2Api
func MakeEc2Api() (Ec2Api, error) {
	Api = Ec2Api{
		Filters: map[string]types.Filter{
			"is-minecraft-server": {
				Name:   aws.String(fmt.Sprintf("tag:%s", MinecraftTagKey)),
				Values: []string{MinecraftTagValue},
			},
		},
	}

	return Api, nil
}

func (api *Ec2Api) GetMinecraftInstances() (ec2.DescribeInstancesOutput, error) {
	var output *ec2.DescribeInstancesOutput
	var err error

	output, err = api.Client.DescribeInstances(api.Context, &ec2.DescribeInstancesInput{
		DryRun: aws.Bool(false),
		Filters: []types.Filter{
			api.Filters["is-minecraft-server"],
		},
	})
	if err != nil {
		return ec2.DescribeInstancesOutput{}, err
	}

	return *output, nil
}

func (api *Ec2Api) GetMinecraftPortMappings() (map[string]string, error) {
	ports := map[string]string{}

	rulesOutput, err := api.Client.DescribeSecurityGroupRules(api.Context, &ec2.DescribeSecurityGroupRulesInput{
		DryRun: aws.Bool(false),
		Filters: []types.Filter{
			api.Filters["is-minecraft-server"],
		},
	})
	if err != nil {
		return map[string]string{}, err
	}

	for _, rules := range rulesOutput.SecurityGroupRules {
		ports[*rules.GroupId] = strconv.FormatInt(int64(*rules.FromPort), 10)
	}

	return ports, err
}

func (api *Ec2Api) GetMinecraftInstancesOutput(instances ec2.DescribeInstancesOutput, ports map[string]string) []MinecraftInstanceOutput {
	var output []MinecraftInstanceOutput

	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			// Get the instance name from the instance tags
			instanceName := ""
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					instanceName = *tag.Value
				}
			}

			// Loop through the security groups
			for _, sg := range instance.SecurityGroups {
				pIp := ""
				if instance.PublicIpAddress != nil {
					pIp = *instance.PublicIpAddress
				}

				output = append(output, MinecraftInstanceOutput{
					Name:   instanceName,
					Id:     *instance.InstanceId,
					Port:   ports[*sg.GroupId],
					Ip:     pIp,
					Status: instance.State.Name,
				})
			}
		}
	}

	return output
}

// GetInstances grab all servers related to minecraft
func (api *Ec2Api) GetInstances() []MinecraftInstanceOutput {
	instances, err := api.GetMinecraftInstances()
	if err != nil {
		log.Fatalln(err)
	}

	ports, err := api.GetMinecraftPortMappings()
	if err != nil {
		log.Fatalln(err)
	}

	return api.GetMinecraftInstancesOutput(instances, ports)
}

func (api *Ec2Api) StartInstance(instanceId string) bool {
	var started bool
	output, err := api.Client.StartInstances(api.Context, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return false
	}

	for _, startedInstance := range output.StartingInstances {
		if *startedInstance.InstanceId == instanceId {
			started = true
		}
	}

	return started
}

func (api *Ec2Api) StopInstance(instanceId string) bool {
	var stopped bool
	output, err := api.Client.StopInstances(api.Context, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return false
	}

	for _, stoppingInstance := range output.StoppingInstances {
		if *stoppingInstance.InstanceId == instanceId {
			stopped = true
		}
	}

	return stopped
}

// SetClient set the ec2 client
func (api *Ec2Api) SetClient() {
	// Create an Amazon S3 service client
	api.Client = ec2.NewFromConfig(api.Config)
}

// SetConfig set the ec2 config
func (api *Ec2Api) SetConfig() error {
	var err error
	api.Config, err = config.LoadDefaultConfig(api.Context, config.WithCredentialsProvider(api.CredentialsProvider))
	if err != nil {
		return err
	}

	return nil
}

// SetCredentials set the aws credentials
func (api *Ec2Api) SetCredentials(i Ec2CredentialsInput) {
	api.CredentialsProvider = credentials.NewStaticCredentialsProvider(i.Id, i.Secret, i.Session)
}

// Setup setup the ec2 api
func (api *Ec2Api) Setup(i Ec2CredentialsInput) error {
	api.Context = context.TODO()
	// Set the credentials
	api.SetCredentials(i)
	// Set the config
	err := api.SetConfig()
	if err != nil {
		return err
	}
	// Set the client
	api.SetClient()

	return nil
}
