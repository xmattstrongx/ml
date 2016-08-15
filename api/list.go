package main

import (
	"encoding/json"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
)

func ListTasks(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO figure out how to get framework to upload config.json to rmeove remove hardcoded creds
	// config := LoadConfig("./config.json")

	sess := session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	params := &dynamodb.ScanInput{
		TableName:       aws.String("task-lists"), // Required
		AttributesToGet: []*string{aws.String("taskid"), aws.String("description"), aws.String("priority"), aws.String("completed"), aws.String("user")},
	}
	resp, err := db.Scan(params)
	if err != nil {
		logger.Errorf("DB Error resp: %+v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var tasks []Task

	if len(resp.Items) > 0 {
		for _, v := range resp.Items {
			tasks = append(tasks, AttributesToTask(v))
		}
	}

	result, _ := json.Marshal(tasks)
	w.Write(result)
}

func GetListLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", ListTasks, &sparta.LambdaFunctionOptions{Description: "RESTful get for tasklist", Timeout: 10})
}
