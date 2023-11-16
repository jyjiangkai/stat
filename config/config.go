package config

type S3 struct {
	AWSAccessKeyID     string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `yaml:"aws_secret_access_key"`
}
