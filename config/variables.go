package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type variables struct {
	AwsS3Bucket        string
	AwsAccessKeyId     string
	AwsSecretAccessKey string
}

var Variables variables

func LoadVariables() error {
	err := godotenv.Load()

	if err == nil {
		log.Println(
			"environmental variables loaded from .env file",
		)
	}

	Variables.AwsS3Bucket = os.Getenv("AWS_S3_BUCKET")
	Variables.AwsAccessKeyId = os.Getenv("AWS_ACCESS_KEY_ID")
	Variables.AwsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	return nil
}
