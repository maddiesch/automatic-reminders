package auto

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrRecordNotFound = errors.New("record not found")

// PrimaryKey contains the compound key for a records primary key
type PrimaryKey struct {
	HashKey string
	SortKey string
}

func (p PrimaryKey) Dynamo() map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"PK": {S: aws.String(p.HashKey)},
		"SK": {S: aws.String(p.SortKey)},
	}
}

type Record interface {
	PrimaryKey() PrimaryKey
}
