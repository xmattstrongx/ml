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
)

// Task struct declaration
type task struct {
	TaskID      string `json:"taskid"`
	User        string `json:"user"`
	Description string `json:"description"`
	Priority    *int   `json:"priority"`
	Completed   string `json:"completed"`
}

func updateTask(event *json.RawMessage,
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

	var task task
	json.Unmarshal(*event, &task)

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
	}

	// if taskid exists continue update operation
	if _, ok := exists.Item["taskid"]; ok {

		// Perform data validation based on requirements. If data validatin fails bail out.
		if err = task.validateTask(); err != nil {
			log.Printf("Bad Request. Error: %v", err)
		} else {
			task.convertTimeToISO()

			params := &dynamodb.UpdateItemInput{
				Key: map[string]*dynamodb.AttributeValue{ // Required
					"taskid": {
						S: aws.String(task.TaskID),
					},
				},
				TableName:        aws.String("task-lists"),
				AttributeUpdates: createDynamoItem(task),
			}
			resp, err := db.UpdateItem(params)
			if err != nil {
				log.Fatal(err.Error())
			}
			log.Printf("Successful response: %+v\n", resp)
			log.Println("Udpate successful!")
		}
	} else {
		log.Printf("Update failed. No item exists with id: %s", task.TaskID)
	}
}

// TODO refactor this madness. Creates an AWS UpdateItemInput AttributeUpdates struct
func createDynamoItem(task task) map[string]*dynamodb.AttributeValueUpdate {
	if task.User == "" && task.Completed == "" {
		return map[string]*dynamodb.AttributeValueUpdate{
			"description": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(task.Description),
				},
			},
			"priority": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
				},
			},
		}
	} else if task.User == "" {
		return map[string]*dynamodb.AttributeValueUpdate{
			"description": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(task.Description),
				},
			},
			"priority": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
				},
			},
			"completed": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(task.Completed),
				},
			},
		}
	} else if task.Completed == "" {
		return map[string]*dynamodb.AttributeValueUpdate{
			"user": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(task.User),
				},
			},
			"description": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(task.Description),
				},
			},
			"priority": {
				Value: &dynamodb.AttributeValue{
					S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
				},
			},
		}
	}
	return map[string]*dynamodb.AttributeValueUpdate{
		"user": {
			Value: &dynamodb.AttributeValue{
				S: aws.String(task.User),
			},
		},
		"description": {
			Value: &dynamodb.AttributeValue{
				S: aws.String(task.Description),
			},
		},
		"priority": {
			Value: &dynamodb.AttributeValue{
				S: aws.String(strconv.Itoa(*task.Priority)), // aws driver doesnt have int property. Deal with it later
			},
		},
		"completed": {
			Value: &dynamodb.AttributeValue{
				S: aws.String(task.Completed),
			},
		},
	}
}

func (t *task) sanitize() {
	if t.TaskID != "" {
		t.TaskID = strings.TrimSpace(t.TaskID)
	}
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

func (t *task) convertTimeToISO() {
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

func (t task) validateTask() error {
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
	lambdaFn := sparta.NewLambda("taskAccessRole", updateTask, &sparta.LambdaFunctionOptions{Description: "RESTful update for tasklist", Timeout: 10})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Deploy it
	sparta.Main("updateTask",
		"Simple Sparta AWS Lambda REST update function",
		lambdaFunctions,
		nil,
		nil)
}
