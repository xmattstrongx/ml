package main

import (
	"encoding/json"
	"log"
	"strconv"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
	"github.com/pborman/uuid"
)

func createTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO remove hardcoded creds
	sess := session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	// Create a DynamoDB client with additional configuration
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	var task Task
	json.Unmarshal(*event, &task)
	if task.Description != "" && task.Priority != nil {
		params := &dynamodb.PutItemInput{
			TableName: aws.String("task-lists"),
			Item: map[string]*dynamodb.AttributeValue{
				"taskid": {
					S: aws.String(uuid.New()),
				},
				"user": {
					S: aws.String(task.User),
				},
				"description": {
					S: aws.String(task.Description),
				},
				"priority": {
					S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
				},
				"completed": {
					S: aws.String(task.Completed),
				},
			},
		}
		log.Printf("%+v\n", params)

		if resp, err := db.PutItem(params); err != nil {
			log.Printf("%+v\n", resp)
			log.Fatal(err.Error())
		} else {
			log.Printf("%+v\n", resp)
		}
		log.Println("Create successful!")
	} else {
		log.Println("Create failed. Be sure description and priority are provided.")
	}
}

func main() {

	// Sparta framework lambda function setup
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.NewLambda(sparta.IAMRoleDefinition{}, createTask, nil)
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("TaskList",
		"Simple Sparta application that creates AWS Lambda functions",
		lambdaFunctions,
		nil,
		nil)
}

type Task struct {
	User        string `json:"user"`
	Description string `json:"description"`
	Priority    *int   `json:"priority"`
	Completed   string `json:"completed"`
}
