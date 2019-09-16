package auto

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/maddiesch/serverless"
)

var (
	dbInstance *dynamodb.DynamoDB
	dbSetup    sync.Once
)

// DynamoDB returns the shared DynamoDB client instance
func DynamoDB() *dynamodb.DynamoDB {
	dbSetup.Do(func() {
		dbInstance = serverless.NewDB("", "").Client
	})
	return dbInstance
}

// TableName returns the DynamoDB table name
func TableName() *string {
	return aws.String(os.Getenv("DYNAMODB_TABLE_NAME"))
}

// FormatString returns the DynamoDB AttributeValue String Value with a format
func FormatString(format string, args ...interface{}) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{S: aws.String(fmt.Sprintf(format, args...))}
}

// DynamoTime returns the DynamoDB AttributeValue for a time
func DynamoTime(t time.Time) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{N: aws.String(fmt.Sprintf("%d", t.Unix()))}
}

// HashString performs a SHA-256 on the passed string and returns the hex-encoded bytes
func HashString(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

// TimeFromDynamo parses and returns the time from a dynamodb attribute value
func TimeFromDynamo(a *dynamodb.AttributeValue) time.Time {
	if a == nil || a.N == nil {
		return time.Time{}
	}

	value, err := strconv.ParseInt(aws.StringValue(a.N), 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(value, 0)
}
