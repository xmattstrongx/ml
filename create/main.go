package main

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	sparta "github.com/mweagle/Sparta"
	"github.com/pborman/uuid"
)

// Task struct declaration
type Task struct {
	User        string `json:"user"`
	Description string `json:"description"`
	Priority    *int   `json:"priority"`
	Completed   string `json:"completed"`
}

func createTask(event *json.RawMessage,
	context *sparta.LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	// TODO remove hardcoded creds
	sess := session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})

	// Create a DynamoDB client with additional configuration
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	var task Task
	json.Unmarshal(*event, &task)

	// trim whitespace
	task.sanitize()
	task.convertTimeToISO()

	// data validation based on requirements
	err := task.validateTask()
	if err != nil {
		log.Printf("Bad Request. Error: %v", err)
	} else {
		params := &dynamodb.PutItemInput{
			TableName: aws.String("task-lists"),
			Item:      createDynamoItem(task),
		}

		// insert new item in to dynamoDB. If that fails bail out.
		resp, err := db.PutItem(params)
		if err != nil {
			log.Printf("In Error resp: %+v\n", resp)
			log.Fatal(err.Error())
		}

		log.Printf("Successful response: %+v\n", resp)
		log.Println("Create successful!")
	}
}

// TODO refactor this madness. Creates a
func createDynamoItem(task Task) map[string]*dynamodb.AttributeValue {
	if task.User == "" && task.Completed == "" {
		return map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(uuid.New()),
			},
			"description": {
				S: aws.String(task.Description),
			},
			"priority": {
				S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
			},
		}
	} else if task.User == "" {
		return map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(uuid.New()),
			},
			"description": {
				S: aws.String(task.Description),
			},
			"priority": {
				S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
			},
			"completed": {
				S: aws.String(task.Completed),
			},
		}
	} else if task.Completed == "" {
		return map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(uuid.New()),
			},
			"user": {
				S: aws.String(task.User),
			},
			"description": {
				S: aws.String(task.Description),
			},
			"priority": {
				S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
			},
		}
	}
	return map[string]*dynamodb.AttributeValue{
		"taskid": {
			S: aws.String(uuid.New()),
		},
		"user": {
			S: aws.String(task.User),
		},
		"description": {
			S: aws.String(task.Description),
		},
		"priority": {
			S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
		},
		"completed": {
			S: aws.String(task.Completed),
		},
	}
}

func (t *Task) sanitize() {
	if t.User != "" {
		t.User = strings.TrimSpace(t.User)
	}
	if t.Description != "" {
		t.Description = strings.TrimSpace(t.Description)
	}
	if t.Completed != "" {
		t.Completed = strings.TrimSpace(t.Completed)
	}
}

func (t *Task) convertTimeToISO() {
	// if completed is provided alter timestamp to ISO8061
	if t.Completed != "" {
		tt, err := time.Parse("20060102T15:04:05-07:00", t.Completed)
		if err != nil {
			log.Printf("Unable to format provided timestamp provided in completed")
			t.Completed = ""
		} else {
			t.Completed = string(tt.Format("2006-01-02T15:04:05-0700"))
		}
	}
}

func (t Task) validateTask() error {
	log.Printf("Task: %v\n", t)

	// If user chooses to provide email then validate 5 <= user <= 254
	if t.User != "" {
		if len(t.User) < 5 || len(t.User) > 254 {
			return errors.New("Length of users email address must be no less than 5 characters and no greater than 254 characters")
		}
	}

	// Validate Description has been provided by the user
	if t.Description == "" {
		return errors.New("Request missing required description")
	}

	// Validate priority has been provided by the user
	if t.Priority == nil {
		return errors.New("Request missing required priority")
	}

	// Validate 0 <= priority <= 9
	if *t.Priority < 0 || *t.Priority > 9 {
		return errors.New("Priority must be greater than 0 and no greater than 9")
	}
	return nil
}

func main() {

	// Sparta framework lambda function setup
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.NewLambda(sparta.IAMRoleDefinition{}, createTask, &sparta.LambdaFunctionOptions{Description: "RESTful create for tasklist", Timeout: 10})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("createTask",
		"Simple Sparta AWS Lambda REST create function",
		lambdaFunctions,
		nil,
		nil)
}
