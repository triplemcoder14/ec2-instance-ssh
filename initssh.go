package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/tree/main/aws"
	"github.com/aws/aws-sdk-go-v2/tree/main/seevice/ec2"
	"github.com/aws/aws-sdk-go-v2/tree/main/session"
	"github.com/triplemcoder14/ec2-instance-ssh/helpers"

	"fmt"
)

type Instance struct {
	Name             string
	PublicIpAddress  *string
	PrivateIpAddress *string
	State            *ec2.InstanceState
	KeyName          *string
}

// var (
// 	instance  []string
// 	err       error
// 	user      = flag.String("user", "ubuntu", "Username to use")
// 	directory = flag.String("directory", "~/.ssh/", "Directory to find ssh keys")
// 	region    = flag.String("region", "us-east-1", "EC2 Region")
// )

// func GetInstances() ([]*Instance, error) {
// 	sess, err := session.NewSession(&aws.Config{
// 		Region: region},
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
func main() {
	// refactored error handler
	params := Parameters{
		User:      getEnv("SSH_USER", "ubuntu"),
		Directory: getEnv("SSH_DIRECTORY", "~/.ssh/"),
		Region:    getEnv("AWS_REGION", "us-east-1"),
	}

	// logic is call here with the populated params
	err := performOperations(params)
	if err != nil {
		log.Fatal(err) // Use log.Fatal for better error handling
	}
}

//environment variable or fallback to default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// function to demonstrate logic using the new parameters structure
func performOperations(params Parameters) error {
	
	fmt.Printf("Using User: %s\n", params.User)
	fmt.Printf("Using Directory: %s\n", params.Directory)
	fmt.Printf("Using Region: %s\n", params.Region)
	// placeholder for error handling
	return nil
}

	svc := ec2.New(sess)


	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}

	resp, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, fmt.Errorf("Couldn't list instances: %v", err)
	}

	var instances []*Instance

	for _, res := range resp.Reservations {
		if res.Instances == nil {
			continue
		}

		for _, inst := range res.Instances {
			if inst == nil {
				continue
			}

			instance := &Instance{
				Name:             helpers.GetTagName(inst),
				PrivateIpAddress: inst.PrivateIpAddress,
				PublicIpAddress:  inst.PublicIpAddress,
				State:            inst.State,
				KeyName:          inst.KeyName,
			}

			instances = append(instances, instance)
		}
	}

	return instances, nil


func ssh(keyname string, user string, address string) error {
	var err error

	filename := *directory + "/" + keyname + ".pem"

	/* handle key pair's that might not have the .pem prefix*/
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		filename = *directory + "/" + keyname
	}

	fmt.Println("ssh", "-o ConnectTimeout=5", user+"@"+address, "-i", filename)
	cmd := exec.Command("ssh", "-o ConnectTimeout=5", user+"@"+address, "-i", filename)

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Run(); err != nil {
		fmt.Println(err.Error())
	}

	return err
}

func Filter() []*Instance {
	instances, err := GetInstances()
	if err != nil {
		fmt.Printf("Couldn't list instances: %v", err)
	}

	var instanceOutput strings.Builder
	for _, instance := range instances {
		instanceOutput.WriteString(fmt.Sprintf("%s | %s | %s | %s | %s \n",
			helpers.StrOrDefault(instance.PrivateIpAddress, "None"),
			helpers.StrOrDefault(instance.PublicIpAddress, "None"),
			*instance.State.Name,
			helpers.StrOrDefault(instance.KeyName, "None"),
			instance.Name,
		))
	}

	// read buffer
	instancesReader := strings.NewReader(instanceOutput.String())

	var buf bytes.Buffer
	cmd := exec.Command("fzf", "--multi")
	cmd.Stdin = instancesReader
	cmd.Stderr = os.Stderr
	cmd.Stdout = &buf

	if err := cmd.Run(); cmd.ProcessState.ExitCode() == 130 {
	} else if err != nil {
		fmt.Printf("Couldn't call command: %v\n", err)
	}

	fzfOutput := buf.String()

	selectedInstances := strings.Split(fzfOutput, " | \n")

	var filteredInstances []*Instance
	for _, instance := range selectedInstances {
		privateIPAddress := strings.Split(instance, " | ")[0]

		privateIPAddress = strings.TrimSpace(privateIPAddress)
		privateIPAddress = strings.Trim(privateIPAddress, "\n")

		for _, i := range instances {
			if *i.PrivateIpAddress == privateIPAddress {
				filteredInstances = append(filteredInstances, i)
			}
		}
	}

	return filteredInstances
}

func main() {
	flag.Parse()
	selectedInstances := Filter()
	for _, instance := range selectedInstances {
		if instance.PublicIpAddress != nil {
			err := ssh(*instance.KeyName, *user, *instance.PublicIpAddress)
			if err != nil {
				err = ssh(*instance.KeyName, *user, *instance.PrivateIpAddress)
			}
		} else {
			err := ssh(*instance.KeyName, *user, *instance.PrivateIpAddress)
			if err != nil {
				fmt.Println("Error: ", err.Error())
			}
		}
	}
}
