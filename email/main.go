package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ses"
	sparta "github.com/mweagle/Sparta"
)

// Task struct declaration
type task struct {
	TaskID      string `json:"taskid"`
	User        string `json:"user"`
	Description string `json:"description"`
	Priority    *int   `json:"priority"`
	Completed   string `json:"completed"`
}

func emailTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	sess := session.New(&aws.Config{
		Region: aws.String("us-west-2"),
		// Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	incompleteTaskList, err := getUncompletedTasks(sess)
	if err != nil {
		log.Println(err.Error())
	}

	var tasks []task

	if len(incompleteTaskList.Items) > 0 {
		for _, v := range incompleteTaskList.Items {
			tasks = append(tasks, attributesToTask(v))
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
		log.Println("No incomplete tasks")
	}
}
func getUncompletedTasks(sess *session.Session) (*dynamodb.ScanOutput, error) {
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	params := &dynamodb.ScanInput{
		TableName:        aws.String("task-lists"),
		FilterExpression: aws.String("completed = :val"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val": {
				S: aws.String("0"),
			},
		},
	}

	taskList, err := db.Scan(params)
	if err != nil {
		return nil, err
	}

	return taskList, nil
}

func attributesToTask(info map[string]*dynamodb.AttributeValue) task {
	var t task
	if value, missing := info["taskid"]; missing {
		t.TaskID = *value.S
	}
	if value, missing := info["user"]; missing {
		t.User = *value.S
	}
	if value, missing := info["description"]; missing {
		t.Description = value.GoString()
	}
	if value, missing := info["priority"]; missing {
		pr, err := strconv.Atoi(*value.S)
		if err == nil {
			t.Priority = &pr
		}
	}
	if value, missing := info["completed"]; missing {
		t.Completed = *value.S
	}
	return t
}

func getSendEmailInput(tasks []task) []ses.SendEmailInput {
	var inputs []ses.SendEmailInput
	for _, val := range tasks {
		//TODO EMAIL VALIDATION
		if val.User != "null" && val.User != "" {
			inputs = append(inputs, ses.SendEmailInput{
				Destination: &ses.Destination{
					ToAddresses: []*string{
						aws.String(val.User),
					},
				},
				Message: &ses.Message{
					Body: &ses.Body{
						Html: &ses.Content{
							Data:    aws.String(fmt.Sprintf("Dear %s, <br/>Task: %s is not yet completed.<br/><br/>Description: %s<br/> Priority: %d", val.User, val.TaskID, val.Description, &val.Priority)),
							Charset: aws.String("us-ascii"),
						},
						Text: &ses.Content{
							//Attempted to use go template but aws/ses package wasnt having it
							Data:    aws.String(fmt.Sprintf("Dear %s, \nTask: %s is not yet completed.\n\nDescription: %s\n Priority: %d", val.User, val.TaskID, val.Description, &val.Priority)),
							Charset: aws.String("us-ascii"),
						},
					},
					Subject: &ses.Content{
						Data:    aws.String(fmt.Sprintf("Incomplete Task: %s", val.TaskID)),
						Charset: aws.String("us-ascii"),
					},
				},
				Source: aws.String("mstrong1341@gmail.com"),
			})
		}
	}
	return inputs
}

func main() {

	// Sparta framework lambda function setup
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.NewLambda("taskAccessRole", emailTask, &sparta.LambdaFunctionOptions{Description: "Email for tasklist", Timeout: 10})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("emailTask",
		"Simple Sparta AWS Lambda REST create function",
		lambdaFunctions,
		nil,
		nil)
}
