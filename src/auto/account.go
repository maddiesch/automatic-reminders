package auto

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/maddiesch/serverless/amazon"
)

const (
	accountRecordSortKey = "_USER_ACCOUNT"
)

// Account represents a user account
type Account struct {
	ID                  string
	FirstName           string
	LastName            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastAuthenticatedAt time.Time
	AutomaticID         string `json:"-"`
}

// FindAccount returns the account with the passed ID.
func FindAccount(accountID string) (*Account, error) {
	item, err := DynamoDB().GetItem(&dynamodb.GetItemInput{
		TableName: TableName(),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(accountID)},
			"SK": {S: aws.String(accountRecordSortKey)},
		},
	})
	if err != nil && amazon.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return &Account{
		ID:                  aws.StringValue(item.Item["PK"].S),
		FirstName:           aws.StringValue(item.Item["FirstName"].S),
		LastName:            aws.StringValue(item.Item["LastName"].S),
		CreatedAt:           TimeFromDynamo(item.Item["CreatedAt"]),
		UpdatedAt:           TimeFromDynamo(item.Item["UpdatedAt"]),
		LastAuthenticatedAt: TimeFromDynamo(item.Item["LastAuthenticatedAt"]),
		AutomaticID:         aws.StringValue(item.Item["AutomaticID"].S),
	}, nil
}

// PrimaryKey returns the primary key for DynamoDB
func (a *Account) PrimaryKey() PrimaryKey {
	return PrimaryKey{
		HashKey: a.ID,
		SortKey: accountRecordSortKey,
	}
}
