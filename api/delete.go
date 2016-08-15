package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
)

func DeleteTask(event *json.RawMessage,
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

	var taskid Taskid
	json.Unmarshal([]byte(lambdaEvent.Body), &taskid)

	// trim whitespace
	if taskid.TaskID != "" {
		taskid.TaskID = strings.TrimSpace(taskid.TaskID)
	}

	// verify item exists before delete
	exists, err := db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(taskid.TaskID),
			},
		},
		TableName: aws.String("task-lists"),
		AttributesToGet: []*string{
			aws.String("taskid"),
		},
	})
	if err != nil {
		logger.Errorf("DB Error resp: %+v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If ID exists in DB continue performing delete
	if _, ok := exists.Item["taskid"]; ok {

		params := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{ // Required
				"taskid": {
					S: aws.String(taskid.TaskID),
				},
			},
			TableName: aws.String("task-lists"),
		}

		//Delete item from dynamoDB. If that fails bail out.
		_, err := db.DeleteItem(params)
		if err != nil {
			logger.Errorf("DB Error resp: %+v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			result, _ := json.Marshal(taskid)
			logger.Info("Delete successful!")
			w.Write(result)
		}
	} else {
		res, _ := json.Marshal(fmt.Sprintf("Cannot delete non-existing id: %s", taskid.TaskID))
		w.Write(res)
	}
}

func GetDeleteLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", DeleteTask, &sparta.LambdaFunctionOptions{Description: "RESTful delete for tasklist", Timeout: 10})
}
