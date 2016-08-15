package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/pborman/uuid"
)

func LoadConfig(path string) Configuration {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Config File Missing. ", err)
	}

	var config Configuration
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatal("Config Parse Error: ", err)
	}

	return config
}

func ValidateTask(t Task) error {
	log.Printf("Task: %v\n", t)

	// If user chooses to provide email then validate 5 <= user <= 254
	if t.User != "" {
		if len(t.User) < 5 || len(t.User) > 254 {
			return fmt.Errorf("Length of users email address must be no less than 5 characters and no greater than 254 characters")
		}
	}

	// Validate Description has been provided by the user
	if t.Description == "" {
		return fmt.Errorf("must provide description")
	}

	// Validate priority has been provided by the user
	if t.Priority == nil {
		return fmt.Errorf("must provide priority")
	}

	// Validate 0 <= priority <= 9
	if *t.Priority < 0 || *t.Priority > 9 {
		return fmt.Errorf("Priority must be greater than 0 and no greater than 9")
	}
	return nil
}

func (t *Task) convertTimeToISO() {
	// if completed is provided alter timestamp to ISO8061
	if t.Completed != "" {
		tt, err := time.Parse("20060102T15:04:05-07:00", t.Completed)
		if err != nil {
			log.Printf("Unable to format provided timestamp provided in completed")
			t.Completed = "0"
		}
		t.Completed = string(tt.Format("2006-01-02T15:04:05-0700"))
	} else {
		t.Completed = "0001-01-01T00:00:00+0000"
	}
}

func (t *Task) sanitize() {
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

func AttributesToTask(info map[string]*dynamodb.AttributeValue) Task {
	var t Task
	if value, missing := info["taskid"]; missing {
		t.TaskID = *value.S
	}
	if value, missing := info["user"]; missing {
		t.User = *value.S
	}
	if value, missing := info["description"]; missing {
		t.Description = *value.S
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

func getIncompletedTasks(sess *session.Session) (*dynamodb.ScanOutput, error) {
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-west-2"))

	params := &dynamodb.ScanInput{
		TableName:        aws.String("task-lists"),
		FilterExpression: aws.String("completed = :val"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val": {
				S: aws.String("0001-01-01T00:00:00+0000"),
			},
		},
	}

	taskList, err := db.Scan(params)
	if err != nil {
		return nil, err
	}

	return taskList, nil
}

func getSendEmailInput(tasks []Task) []ses.SendEmailInput {
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

// TODO refactor this madness. Creates an AWS PutItem Item struct
func CreateDynamoDBAttributeValue(task Task) map[string]*dynamodb.AttributeValue {
	if task.User == "" {
		return map[string]*dynamodb.AttributeValue{
			"taskid": {
				S: aws.String(uuid.New()),
			},
			"user": {
				S: aws.String("null"),
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

// TODO refactor this madness. Creates an AWS UpdateItemInput AttributeUpdates struct
func CreateDynamoDBAttributeValueUpdate(task Task) map[string]*dynamodb.AttributeValueUpdate {
	if task.User == "" {
		return map[string]*dynamodb.AttributeValueUpdate{
			"user": {
				Value: &dynamodb.AttributeValue{
					S: aws.String("null"),
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
