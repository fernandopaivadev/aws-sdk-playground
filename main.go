package main

import (
	"encoding/json"
	"fmt"
	"log"

	"main/config"

	"main/application"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func printStruct(data any) (string, error) {
	result, err := json.MarshalIndent(data, "", "    ")

	if err != nil {
		return "", err
	}

	return string(result), nil
}

func main() {
	err := config.LoadVariables()

	if err != nil {
		log.Fatalln("Error loading environment variables >> ", err)
		return
	}

	application.S3.NewClient(s3.Options{
		Region: "sa-east-1",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			config.Variables.AwsAccessKeyId,
			config.Variables.AwsSecretAccessKey,
			""),
		),
	})

	objects, err := application.S3.ListObjects(config.Variables.AwsS3Bucket)

	if err != nil {
		log.Fatalln("Error listing objects >> ", err)
	}

	for i := 0; i < len(objects); i++ {
		fmt.Println(printStruct(objects[i]))
	}
}
