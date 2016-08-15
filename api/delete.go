package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	taskid.sanitize()

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
		log.Fatal(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
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
		resp, err := db.DeleteItem(params)
		if err != nil {
			log.Printf("In Error resp: %+v\n", resp)
			log.Fatal(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			result, _ := json.Marshal(taskid)
			log.Println("Delete successful!")
			w.WriteHeader(http.StatusAccepted)
			w.Write(result)
		}

	} else {
		fmt.Fprintf(w, "Cannot delete non-existing id: %s", taskid.TaskID)
	}
}

func (t *Taskid) sanitize() {
	if t.TaskID != "" {
		t.TaskID = strings.TrimSpace(t.TaskID)
	}
}

func GetDeleteLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", DeleteTask, &sparta.LambdaFunctionOptions{Description: "RESTful delete for tasklist", Timeout: 10})
}
