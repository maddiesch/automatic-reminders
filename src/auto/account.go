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

// DynamoItem returns the DynamoDB AttributeValues for the account
func (a *Account) DynamoItem() (map[string]*dynamodb.AttributeValue, error) {
	pk := a.PrimaryKey()

	return map[string]*dynamodb.AttributeValue{
		"PK":                  {S: aws.String(pk.HashKey)},
		"SK":                  {S: aws.String(pk.SortKey)},
		"GSI2PK":              FormatString("automatic/%s", a.AutomaticID),
		"GSI2SK":              {S: aws.String(AutomaticIndexSortKeyValue)},
		"FirstName":           {S: aws.String(a.FirstName)},
		"LastName":            {S: aws.String(a.LastName)},
		"AutomaticID":         {S: aws.String(a.AutomaticID)},
		"CreatedAt":           DynamoTime(a.CreatedAt),
		"UpdatedAt":           DynamoTime(time.Now()),
		"LastAuthenticatedAt": DynamoTime(time.Now()),
	}, nil
}

// WriteAccountWithToken saves an account & token. Also lets you save other objects alongside in the transaction.
func WriteAccountWithToken(a *Account, t *AutomaticAccessToken, fn func(*Account, *AutomaticAccessToken) []*dynamodb.TransactWriteItem) error {
	primaryKey := a.PrimaryKey()

	accountItem, err := a.DynamoItem()
	if err != nil {
		return err
	}

	tokenItem, err := t.DynamoItem(primaryKey)
	if err != nil {
		return err
	}

	items := []*dynamodb.TransactWriteItem{
		&dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: TableName(),
				Item:      accountItem,
			},
		},
		&dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: TableName(),
				Item:      tokenItem,
			},
		},
	}

	if fn != nil {
		for _, item := range fn(a, t) {
			items = append(items, item)
		}
	}

	_, err = DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})

	return err
}
