package auto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/maddiesch/serverless/amazon"
	"github.com/segmentio/ksuid"
)

type UpdateAccountInput struct {
	AccountID          string
	UpdateStateMachine string
}

func UpdateAccount(input UpdateAccountInput) error {
	payload, err := json.Marshal(map[string]interface{}{
		"AccountID": input.AccountID,
	})
	if err != nil {
		return err
	}

	client := sfn.New(amazon.BaseSession())
	_, err = client.StartExecution(&sfn.StartExecutionInput{
		Name:            aws.String(fmt.Sprintf("update-%s", ksuid.New().String())),
		Input:           aws.String(string(payload)),
		StateMachineArn: aws.String(input.UpdateStateMachine),
	})
	return err
}

func UpdateTripsForAccountUpdate(event *UpdateAccountEvent) error {
	account, err := FindAccount(event.AccountID)
	if err != nil {
		return updateFailureForError(err)
	}

	token, err := FindAutomaticAccessTokenForAccount(account)
	if err != nil {
		return updateFailureForError(err)
	}

	var uri *url.URL
	if event.Context.Current == "STARTING" {
		uri = AutomaticAPIURL("/trip/")
		uri.RawQuery = strings.Join([]string{
			"limit=100",
			"started_at__gte=1325376000",
			fmt.Sprintf("started_at__lte=%d", time.Now().Unix()),
		}, "&")
	} else {
		parsed, err := url.Parse(event.Context.Current)
		if err != nil {
			return updateFailureForError(err)
		}
		uri = parsed
	}

	next, err := UpdateTripsForAccount(account, token, uri)
	if err != nil {
		return updateFailureForError(err)
	}
	if next != nil && next.String() != "" {
		event.Context.Current = next.String()
	} else {
		event.Context.Current = "DONE"
	}

	return nil
}

// UpdateCurrentUpdateState makes a best effort to update the current event's state.
func UpdateCurrentUpdateState(state string, event UpdateAccountEvent) {
	if event.UpdateID == "" {
		return
	}

	DynamoDB().UpdateItem(&dynamodb.UpdateItemInput{
		TableName: TableName(),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(event.UpdateID)},
			"SK": {S: aws.String("_UPDATE_ACCOUNT_")},
		},
		ConditionExpression: aws.String("CurrentState NE :finished"),
		UpdateExpression:    aws.String("SET CurrentState = :state"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":state":    {S: aws.String(state)},
			":finished": {S: aws.String("COMPLETED")},
		},
	})
}

// FindUpdate fetches the existing update from DynamoDB
func FindUpdate(event UpdateAccountEvent) (*AccountUpdate, error) {
	if event.UpdateID == "" {
		return nil, &UpdateFailure{
			Reason: "Can't find an updated without an ID",
			Code:   "EVENT_MISSING_UPDATE_ID",
		}
	}
	output, err := DynamoDB().GetItem(&dynamodb.GetItemInput{
		TableName: TableName(),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(event.UpdateID)},
			"SK": {S: aws.String("_UPDATE_ACCOUNT_")},
		},
	})
	if err != nil && amazon.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, &UpdateFailure{
			Reason: "Can't find an updated without an ID",
			Code:   "EVENT_MISSING_UPDATE_ID",
		}
	} else if err != nil {
		return nil, updateFailureForError(err)
	}

	fmt.Println(output)

	return nil, errors.New("WORKING_ON_IT")
}

// FinishAccountUpdate handles updating the update and unlocking the account for another update
func FinishAccountUpdate(event UpdateAccountEvent) error {
	account, err := FindAccount(event.AccountID)
	if err != nil {
		return updateFailureForError(err)
	}

	items := []*dynamodb.TransactWriteItem{
		&dynamodb.TransactWriteItem{
			Update: &dynamodb.Update{
				TableName:        TableName(),
				Key:              account.PrimaryKey().Dynamo(),
				UpdateExpression: aws.String("REMOVE RunningUpdateID SET AccountUpdatedAt = :time"),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":time": DynamoTime(time.Now()),
				},
			},
		},
	}
	if event.UpdateID != "" {
		duration := time.Since(time.Unix(event.StartTime, 0)) / time.Millisecond
		items = append(items,
			&dynamodb.TransactWriteItem{
				Update: &dynamodb.Update{
					TableName: TableName(),
					Key: map[string]*dynamodb.AttributeValue{
						"PK": {S: aws.String(event.UpdateID)},
						"SK": {S: aws.String("_UPDATE_ACCOUNT_")},
					},
					UpdateExpression: aws.String("SET CompletedAt = :time, UpdateDurationMS = :duration, CurrentState = :state, ExpiresAt = :expires"),
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":time":     DynamoTime(time.Now()),
						":duration": {S: aws.String(fmt.Sprintf("%d", int64(duration)))},
						":state":    {S: aws.String("COMPLETED")},
						":expires":  DynamoTime(time.Now().AddDate(0, 0, 14)),
					},
				},
			})
	}

	_, err = DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})
	if err != nil {
		return updateFailureForError(err)
	}

	return nil
}

// CreateAccountUpdate creates an update for an account.
func CreateAccountUpdate(accountID string) (string, error) {
	account, err := FindAccount(accountID)
	if err != nil {
		return "", updateFailureForError(err)
	}
	updateID := fmt.Sprintf("uid:%s", ksuid.New().String())
	update := &dynamodb.Update{
		TableName:           TableName(),
		ConditionExpression: aws.String("attribute_not_exists(RunningUpdateID)"),
		Key:                 account.PrimaryKey().Dynamo(),
		UpdateExpression:    aws.String("SET RunningUpdateID = :upid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":upid": {S: aws.String(updateID)},
		},
	}
	put := &dynamodb.Put{
		TableName: TableName(),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":           {S: aws.String(updateID)},
			"SK":           {S: aws.String("_UPDATE_ACCOUNT_")},
			"GSI2PK":       {S: aws.String(fmt.Sprintf("account-update/%s", accountID))},
			"GSI2SK":       {S: aws.String(fmt.Sprintf("update/%s", updateID))},
			"CreatedAt":    DynamoTime(time.Now()),
			"CurrentState": {S: aws.String("STARTING")},
		},
	}

	_, err = DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{Put: put},
			{Update: update},
		},
	})
	if err != nil && amazon.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return "", &UpdateFailure{
			Reason: "There is already an update running for this account.",
			Code:   "UPDATE_ALREADY_IN_PROGRESS",
		}
	} else if err != nil {
		return "", updateFailureForError(err)
	}

	return updateID, nil
}

type AccountUpdate struct {
	AccountID string
	UpdateID  string
	StartedAt time.Time
	State     string
}

type UpdateAccountEvent struct {
	AccountID      string
	UpdateID       string
	StartTime      int64
	Context        UpdateAccountContext
	FailureContext *FailureContext
}

type UpdateAccountContext struct {
	Current string
}

type FailureContext struct {
	Error string
	Cause string
}

// UpdateFailure is an error that is handled by the state machine
type UpdateFailure struct {
	Reason string
	Code   string
	Err    error
}

func (e *UpdateFailure) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Reason)
}

// UpdateFailureForError returns an unknown error UpdateFailure
func updateFailureForError(err error) *UpdateFailure {
	return &UpdateFailure{
		Reason: "Unhandled error",
		Code:   "UNHANDLED_ERROR",
		Err:    err,
	}
}
