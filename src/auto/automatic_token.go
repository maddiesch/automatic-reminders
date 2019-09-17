package auto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/segmentio/ksuid"
)

type AutomaticAccessToken struct {
	ID           string    `json:"-"`
	IssuedAt     time.Time `json:"-"`
	UserID       string    `json:"user_id" validate:"required"`
	AccessToken  string    `json:"access_token" validate:"required"`
	ExpiresIn    int       `json:"expires_in" validate:"required"`
	Scope        string    `json:"scope" validate:"required"`
	RefreshToken string    `json:"refresh_token" validate:"required"`
	TokenType    string    `json:"token_type" validate:"required"`
}

func FindAutomaticAccessTokenForAccount(account *Account) (*AutomaticAccessToken, error) {
	query := &dynamodb.QueryInput{
		TableName:              TableName(),
		KeyConditionExpression: aws.String("#pk = :pk AND begins_with(#sk, :sk)"),
		ExpressionAttributeNames: map[string]*string{
			"#pk": aws.String("PK"),
			"#sk": aws.String("SK"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(account.PrimaryKey().HashKey)},
			":sk": {S: aws.String("access-token/")},
		},
		Limit:            aws.Int64(10),
		ScanIndexForward: aws.Bool(false),
	}

	output, err := DynamoDB().Query(query)
	if err != nil {
		return nil, err
	}

	var token *AutomaticAccessToken
	token = nil
	errs := []string{}
	for _, item := range output.Items {
		tmp := &AutomaticAccessToken{}
		err := tmp.UnmarshalDynamoDB(item)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		if tmp.Expired() {
			new, err := tmp.Refresh(account)
			if err != nil {
				continue
			}
			token = new
		} else {
			token = tmp
		}
		if token == nil {
			break
		}
	}
	if token == nil {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return token, nil
}

// Expired checks if the token is expired and should be refreshed.
func (t *AutomaticAccessToken) Expired() bool {
	if t.ExpiresIn == 0 || t.IssuedAt.Unix() == 0 {
		return true
	}

	return t.IssuedAt.Add(time.Duration(t.ExpiresIn) * time.Second).Add(-5 * time.Minute).Before(time.Now())
}

// Refresh performs the refresh. If it succeeds, it writes the new token into DynamoDB and updates the reference to point at the new record
func (t *AutomaticAccessToken) Refresh(account *Account) (*AutomaticAccessToken, error) {
	if t.RefreshToken == "" {
		return nil, errors.New("unable to refresh without a refresh token")
	}

	body, err := json.Marshal(map[string]interface{}{
		"client_id":     Secrets().ClientID,
		"client_secret": Secrets().ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": t.RefreshToken,
	})
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", AutomaticAccountsURL("/oauth/access_token/").String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	response, err := SendRequest(req)
	if err != nil {
		return nil, err
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	token := &AutomaticAccessToken{
		ID:       ksuid.New().String(),
		IssuedAt: time.Now(),
	}

	err = json.Unmarshal(body, token)
	if err != nil {
		return nil, err
	}

	pk := account.PrimaryKey()
	newItem, err := token.DynamoItem(pk)
	if err != nil {
		return nil, err
	}

	items := []*dynamodb.TransactWriteItem{
		&dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: TableName(),
				Item:      newItem,
			},
		},
		&dynamodb.TransactWriteItem{
			Delete: &dynamodb.Delete{
				TableName: TableName(),
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {S: aws.String(pk.HashKey)},
					"SK": FormatString("access-token/%s", t.ID),
				},
			},
		},
	}

	_, err = DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

// SignRequest adds the authorization header to the request
func (t *AutomaticAccessToken) SignRequest(r *http.Request) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.AccessToken))
}

// UnmarshalDynamoDB populates the values from the given attribute value map into the token
func (t *AutomaticAccessToken) UnmarshalDynamoDB(input map[string]*dynamodb.AttributeValue) error {
	t.ID = strings.SplitN(aws.StringValue(input["SK"].S), "/", 2)[1]
	t.IssuedAt = TimeFromDynamo(input["IssuedAt"])
	t.RefreshToken = aws.StringValue(input["RefreshToken"].S)
	t.AccessToken = aws.StringValue(input["AccessToken"].S)
	t.Scope = strings.Join(aws.StringValueSlice(input["Scopes"].SS), " ")
	t.ExpiresIn = int(IntFromDynamo(input["ExpiresIn"]))
	t.TokenType = aws.StringValue(input["TokenType"].S)
	t.UserID = aws.StringValue(input["UserID"].S)

	return nil
}

// DynamoItem marshals the date into DynamoDB item format
func (t *AutomaticAccessToken) DynamoItem(pk PrimaryKey) (map[string]*dynamodb.AttributeValue, error) {
	return map[string]*dynamodb.AttributeValue{
		"PK":           {S: aws.String(pk.HashKey)},
		"SK":           FormatString("access-token/%s", t.ID),
		"ExpiresAt":    DynamoTime(t.IssuedAt.Add(time.Duration(t.ExpiresIn)/time.Second).AddDate(0, 0, 90)),
		"ExpiresIn":    {N: aws.String(fmt.Sprintf("%d", t.ExpiresIn))},
		"Scopes":       {SS: aws.StringSlice(strings.Split(t.Scope, " "))},
		"AccessToken":  {S: aws.String(t.AccessToken)},
		"RefreshToken": {S: aws.String(t.RefreshToken)},
		"IssuedAt":     DynamoTime(time.Now().Add(10 * time.Second)),
		"TokenType":    {S: aws.String(t.TokenType)},
		"UserID":       {S: aws.String(t.UserID)},
		"GSI2PK":       FormatString("access_token/%s", t.UserID),
		"GSI2SK":       FormatString("token-for/%s", pk.HashKey),
	}, nil
}
