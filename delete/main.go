package main

import (
	"encoding/json"
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

// Task struct declaration
type taskid struct {
	TaskID string `json:"taskid"`
}

func deleteTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO remove hardcoded creds
	sess := session.New(&aws.Config{
		Region: aws.String("us-west-2"),
		// Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	// Create a DynamoDB client with additional configuration
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	var taskid taskid
	json.Unmarshal(*event, &taskid)

	// trim whitespace
	taskid.sanitize()

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
	}

	log.Printf("Successful response: %+v\n", resp)
	log.Println("Delete successful!")

}

func (t *taskid) sanitize() {
	if t.TaskID != "" {
		t.TaskID = strings.TrimSpace(t.TaskID)
	}
}

func main() {

	// Sparta framework lambda function setup
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.NewLambda("taskAccessRole", deleteTask, &sparta.LambdaFunctionOptions{Description: "RESTful delete for tasklist", Timeout: 10})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("deleteTask",
		"Simple Sparta AWS Lambda REST delete function",
		lambdaFunctions,
		nil,
		nil)
}
