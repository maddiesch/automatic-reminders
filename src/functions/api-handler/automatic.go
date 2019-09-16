package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
	"github.com/segmentio/ksuid"
)

const (
	automaticIndexSortKeyValue = "_AUTOMATIC_ACCOUNT"
)

func automaticAccountsURL(path string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "accounts.automatic.com",
		Path:   path,
	}
}

func automaticAPIURL(path string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "api.automatic.com",
		Path:   path,
	}
}

func automaticAPISignedRequest(method, path, token string, body interface{}) (*http.Request, error) {
	payload := bytes.NewBuffer(nil)
	if body != nil {
		if bodyBytes, ok := body.([]byte); ok {
			payload = bytes.NewBuffer(bodyBytes)
		} else {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			payload = bytes.NewBuffer(bodyBytes)
		}
	}

	request, err := http.NewRequest(method, automaticAPIURL(path).String(), payload)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if payload != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	if err != nil {
		return nil, err
	}
	return request, nil
}

func integrationAutomaticAuthHandler(c *gin.Context) {
	uri, err := integrationCreateAutomaticAuthenticationURL()
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, uri)
}

func integrationCreateAutomaticAuthenticationURL() (string, error) {
	state := ksuid.New().String()

	input := &dynamodb.PutItemInput{
		TableName: auto.TableName(),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":        auto.FormatString("integration/automatic/%s", state),
			"SK":        {S: aws.String("_REQUEST")},
			"StartedAt": auto.DynamoTime(time.Now()),
			"ExpiresAt": auto.DynamoTime(time.Now().AddDate(0, 0, 2)),
		},
	}

	_, err := auto.DynamoDB().PutItem(input)
	if err != nil {
		return "", err
	}

	scopes := []string{
		"scope:public",
		"scope:user:profile",
		"scope:vehicle:profile",
		"scope:trip",
	}

	values := []string{
		fmt.Sprintf("client_id=%s", auto.Secrets().ClientID),
		fmt.Sprintf("response_type=code"),
		fmt.Sprintf("scope=%s", strings.Join(scopes, "%20")),
		fmt.Sprintf("state=%s", state),
	}

	uri := automaticAccountsURL("/oauth/authorize/")
	uri.RawQuery = strings.Join(values, "&")

	return uri.String(), nil
}

func integrationAutomaticAuthCallbackHandler(c *gin.Context) {
	token, err := integrationAutomaticAuthCallback(c.DefaultQuery("code", ""), c.DefaultQuery("state", ""))
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Token": token})
}

func integrationAutomaticAuthCallback(code, state string) (string, error) {
	if state == "" || code == "" {
		return "", &Error{Status: http.StatusBadRequest, Detail: "Missing state or code"}
	}

	requestKey := map[string]*dynamodb.AttributeValue{
		"PK": auto.FormatString("integration/automatic/%s", state),
		"SK": {S: aws.String("_REQUEST")},
	}

	item, err := auto.DynamoDB().GetItem(&dynamodb.GetItemInput{
		TableName:            auto.TableName(),
		Key:                  requestKey,
		ProjectionExpression: aws.String("StartedAt"),
	})
	if err != nil {
		return "", err
	}
	if len(item.Item) == 0 {
		return "", &Error{
			Status: http.StatusNotFound,
			Detail: "Failed to find a valid authentication request",
		}
	}

	payload, _ := json.Marshal(map[string]string{
		"client_id":     auto.Secrets().ClientID,
		"client_secret": auto.Secrets().ClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
	})

	req, err := http.NewRequest("POST", automaticAccountsURL("/oauth/access_token/").String(), bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return "", err
	}

	response, err := sendRequest(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	token := auto.AutomaticAccessToken{}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return "", err
	}

	result, err := auto.DynamoDB().Query(&dynamodb.QueryInput{
		TableName:              auto.TableName(),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("#pk = :pk AND #sk = :sk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": auto.FormatString("automatic/%s", token.UserID),
			":sk": {S: aws.String(automaticIndexSortKeyValue)},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pk": aws.String("GSI2PK"),
			"#sk": aws.String("GSI2SK"),
		},
	})
	if err != nil {
		return "", err
	}

	var account *auto.Account
	if len(result.Items) == 1 {
		existingAccount, err := integrationAutomaticAuthExistingAccount(token, result.Items[0])
		if err != nil {
			return "", err
		}
		account = existingAccount
	} else {
		newAccount, err := integrationAutomaticAuthCreateAccount(token)
		if err != nil {
			return "", err
		}
		account = newAccount
	}

	if account == nil {
		return "", errors.New("Failed to materialize account")
	}

	auto.DynamoDB().DeleteItem(&dynamodb.DeleteItemInput{
		TableName: auto.TableName(),
		Key:       requestKey,
	})

	return apiTokenForAccount(account)
}

func integrationAutomaticAuthExistingAccount(token auto.AutomaticAccessToken, item map[string]*dynamodb.AttributeValue) (*auto.Account, error) {
	account, err := auto.FindAccount(aws.StringValue(item["PK"].S))
	if err != nil {
		return nil, err
	}

	err = integrationAutomaticWriteAccountInformation(account, nil, token)
	if err != nil {
		return nil, err
	}

	return account, nil
}

type automaticUserStructure struct {
	Username      string `json:"username"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func integrationAutomaticAuthCreateAccount(token auto.AutomaticAccessToken) (*auto.Account, error) {
	request, err := automaticAPISignedRequest("GET", fmt.Sprintf("/user/%s", token.UserID), token.AccessToken, nil)
	if err != nil {
		return nil, err
	}

	response, err := sendRequest(request)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	user := &automaticUserStructure{}
	err = json.Unmarshal(body, user)
	if err != nil {
		return nil, err
	}

	createdTime := time.Now()

	account := &auto.Account{
		ID:          fmt.Sprintf("auid:%s", ksuid.New().String()),
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		CreatedAt:   createdTime,
		UpdatedAt:   createdTime,
		AutomaticID: token.UserID,
	}

	err = integrationAutomaticWriteAccountInformation(account, user, token)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func integrationAutomaticWriteAccountInformation(account *auto.Account, user *automaticUserStructure, token auto.AutomaticAccessToken) error {
	primaryKey := account.PrimaryKey()

	items := []*dynamodb.TransactWriteItem{
		&dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: auto.TableName(),
				Item: map[string]*dynamodb.AttributeValue{
					"PK":                  {S: aws.String(primaryKey.HashKey)},
					"SK":                  {S: aws.String(primaryKey.SortKey)},
					"GSI2PK":              auto.FormatString("automatic/%s", token.UserID),
					"GSI2SK":              {S: aws.String(automaticIndexSortKeyValue)},
					"FirstName":           {S: aws.String(account.FirstName)},
					"LastName":            {S: aws.String(account.LastName)},
					"AutomaticID":         {S: aws.String(token.UserID)},
					"CreatedAt":           auto.DynamoTime(account.CreatedAt),
					"UpdatedAt":           auto.DynamoTime(time.Now()),
					"LastAuthenticatedAt": auto.DynamoTime(time.Now()),
				},
			},
		},
		&dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: auto.TableName(),
				Item: map[string]*dynamodb.AttributeValue{
					"PK":           {S: aws.String(primaryKey.HashKey)},
					"SK":           auto.FormatString("access-token/%s", ksuid.New().String()),
					"ExpiresAt":    auto.DynamoTime(time.Now().Add(time.Duration(token.ExpiresIn)/time.Second).AddDate(0, 0, 90)),
					"ExpiresIn":    {N: aws.String(fmt.Sprintf("%d", token.ExpiresIn))},
					"Scopes":       {SS: aws.StringSlice(strings.Split(token.Scope, " "))},
					"AccessToken":  {S: aws.String(token.AccessToken)},
					"RefreshToken": {S: aws.String(token.RefreshToken)},
					"GSI1PK":       auto.FormatString("access_token/%s", token.UserID),
					"GSI2SK":       auto.FormatString("token-for/%s", primaryKey.HashKey),
				},
			},
		},
	}

	if user != nil {
		items = append(items, &dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: auto.TableName(),
				Item: map[string]*dynamodb.AttributeValue{
					"PK":             {S: aws.String(primaryKey.HashKey)},
					"SK":             auto.FormatString("contact/%s/_EMAIL", auto.HashString(user.Email)),
					"ContactValue":   {S: aws.String(user.Email)},
					"ContactType":    {S: aws.String("EMAIL")},
					"ReceiveContact": {N: aws.String("1")},
				},
			},
		})
	}

	_, err := auto.DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})

	return err
}
