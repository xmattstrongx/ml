package main

import (
	"encoding/json"
	"fmt"

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
		logger.Error("Start Request", "Bad request: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// if taskid exists continue update operation
	if _, ok := exists.Item["taskid"]; ok {

		// Perform data validation based on requirements. If data validatin fails bail out.
		if err = ValidateTask(task); err != nil {
			logger.Errorf("Bad Request. Error: %v", err)
			fmt.Fprintf(w, "Error: %s", err.Error())
		} else {
			task.Completed = ConvertTimeToISO(task.Completed)

			params := &dynamodb.UpdateItemInput{
				Key: map[string]*dynamodb.AttributeValue{ // Required
					"taskid": {
						S: aws.String(task.TaskID),
					},
				},
				TableName:        aws.String("task-lists"),
				AttributeUpdates: CreateDynamoDBAttributeValueUpdate(task),
			}
			_, err := db.UpdateItem(params)
			if err != nil {
				logger.Errorf("DB Error resp: %+v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				logger.Info("Update successful!")
				result, _ := json.Marshal(task)
				w.Write(result)
			}
		}
	} else {
		res, _ := json.Marshal(fmt.Sprintf("No item to update with taskid: %s", task.TaskID))
		w.Write(res)
	}
}

func GetUpdateLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", UpdateTask, &sparta.LambdaFunctionOptions{Description: "RESTful update for tasklist", Timeout: 10})
}
