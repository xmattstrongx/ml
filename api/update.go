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

func UpdateTask(event *json.RawMessage,
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

	// Verify taskID exists in database before proceeding
	exists, err := db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(task.TaskID),
			},
		},
		TableName: aws.String("task-lists"),
		AttributesToGet: []*string{
			aws.String("taskid"),
		},
	})
	if err != nil {
		log.Fatal(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	// if taskid exists continue update operation
	if _, ok := exists.Item["taskid"]; ok {

		// Perform data validation based on requirements. If data validatin fails bail out.
		if err = ValidateTask(task); err != nil {
			log.Printf("Bad Request. Error: %v", err)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "Error: %s", err.Error())
		} else {
			task.convertTimeToISO()

			params := &dynamodb.UpdateItemInput{
				Key: map[string]*dynamodb.AttributeValue{ // Required
					"taskid": {
						S: aws.String(task.TaskID),
					},
				},
				TableName:        aws.String("task-lists"),
				AttributeUpdates: CreateDynamoDBAttributeValueUpdate(task),
			}
			resp, err := db.UpdateItem(params)
			if err != nil {
				log.Printf("In Error resp: %+v\n", resp)
				log.Fatal(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				log.Println("Update successful!")
				result, _ := json.Marshal(task)
				w.WriteHeader(http.StatusCreated)
				w.Write(result)
			}
		}
	} else {
		fmt.Fprintf(w, "Update failed. No item exists with id: %s", task.TaskID)
	}
}

func GetUpdateLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", UpdateTask, &sparta.LambdaFunctionOptions{Description: "RESTful update for tasklist", Timeout: 10})
}
