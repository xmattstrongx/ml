package main

import (
	"encoding/json"
	"log"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
)

func listTasks(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO remove hardcoded creds
	sess := session.New(&aws.Config{
		Region: aws.String("us-west-2"),
		// Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	params := &dynamodb.ScanInput{
		TableName:       aws.String("task-lists"), // Required
		AttributesToGet: []*string{aws.String("taskid"), aws.String("description"), aws.String("priority"), aws.String("completed"), aws.String("user")},
	}
	resp, err := db.Scan(params)
	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println(resp)
	}
}

func main() {

	// Sparta framework lambda function setup
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.NewLambda("taskAccessRole", listTasks, &sparta.LambdaFunctionOptions{Description: "RESTful list for tasklist", Timeout: 10})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("listTasks",
		"Simple Sparta AWS Lambda REST list function",
		lambdaFunctions,
		nil,
		nil)
}
