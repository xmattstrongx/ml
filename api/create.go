package main

import (
	"encoding/json"
	"fmt"
	"log"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
)

type Lamdba struct {
	lambda func()
	sess   *session.Session
}

func CreateTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO figure out how to get framework to upload config.json to rmeove remove hardcoded creds
	// config := LoadConfig("./config.json")

	sess := session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	// Create a DynamoDB client with additional configuration
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	var lambdaEvent sparta.APIGatewayLambdaJSONEvent
	err := json.Unmarshal([]byte(*event), &lambdaEvent)

	var task Task
	json.Unmarshal([]byte(lambdaEvent.Body), &task)

	// trim whitespace
	task.sanitize()

	task.convertTimeToISO()

	// Data validation based on requirements. If fails bail out.
	err = ValidateTask(task)
	if err != nil {
		log.Printf("Bad Request. Error: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Error: %s", err.Error())
	} else {
		params := &dynamodb.PutItemInput{
			TableName: aws.String("task-lists"),
			Item:      CreateDynamoDBAttributeValue(task),
		}

		// insert new item in to dynamoDB. If that fails bail out.
		resp, err := db.PutItem(params)
		if err != nil {
			log.Printf("In Error resp: %+v\n", resp)
			log.Fatal(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			log.Println("Create successful!")
			result, _ := json.Marshal(task)
			w.WriteHeader(http.StatusCreated)
			w.Write(result)
		}
	}
}

func GetCreateLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", CreateTask, &sparta.LambdaFunctionOptions{Description: "RESTful create for tasklist", Timeout: 10})
}
