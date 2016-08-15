package main

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	sparta "github.com/mweagle/Sparta"
)

func EmailTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO figure out how to get framework to upload config.json to rmeove remove hardcoded creds
	// config := LoadConfig("./config.json")

	sess := session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	incompleteTaskList, err := getIncompletedTasks(sess)
	if err != nil {
		logger.Error(err)
	}

	var tasks []Task

	if len(incompleteTaskList.Items) > 0 {
		for _, v := range incompleteTaskList.Items {
			tasks = append(tasks, AttributesToTask(v))
		}

		emailParams := getSendEmailInput(tasks)
		svc := ses.New(sess)

		for _, val := range emailParams {
			resp, err := svc.SendEmail(&val)
			if err != nil {
				logger.Error(err)
			} else {
				logger.Info(resp)
			}
		}
	} else {
		logger.Info("No incomplete tasks")
	}
}

func GetEmailLambda() *sparta.LambdaAWSInfo {
	return sparta.NewLambda("taskAccessRole", EmailTask, &sparta.LambdaFunctionOptions{Description: "Email job for tasklist", Timeout: 10})
}
