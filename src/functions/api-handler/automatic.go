package main

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
	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
	"github.com/maddiesch/serverless"
	"github.com/segmentio/ksuid"
)

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

	request, err := http.NewRequest(method, auto.AutomaticAPIURL(path).String(), payload)
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

	uri := auto.AutomaticAccountsURL("/oauth/authorize/")
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

	req, err := http.NewRequest("POST", auto.AutomaticAccountsURL("/oauth/access_token/").String(), bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return "", err
	}

	response, err := auto.SendRequest(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	token := auto.AutomaticAccessToken{
		ID:       ksuid.New().String(),
		IssuedAt: time.Now(),
	}

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
			":sk": {S: aws.String(auto.AutomaticIndexSortKeyValue)},
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

	response, err := auto.SendRequest(request)
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
	return auto.WriteAccountWithToken(account, &token, func(a *auto.Account, t *auto.AutomaticAccessToken) []*dynamodb.TransactWriteItem {
		if user == nil {
			return []*dynamodb.TransactWriteItem{}
		}
		return []*dynamodb.TransactWriteItem{
			&dynamodb.TransactWriteItem{
				Put: &dynamodb.Put{
					TableName: auto.TableName(),
					Item: map[string]*dynamodb.AttributeValue{
						"PK":             {S: aws.String(a.PrimaryKey().HashKey)},
						"SK":             auto.FormatString("contact/%s/_EMAIL", auto.HashString(user.Email)),
						"ContactValue":   {S: aws.String(user.Email)},
						"ContactType":    {S: aws.String("EMAIL")},
						"ReceiveContact": {N: aws.String("1")},
					},
				},
			},
		}
	})
}

func integrationAutomaticHookshotHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		reportError(err, false)
		respondWithError(c, err)
		return
	}

	serverless.Log(string(body))

	c.Status(http.StatusNoContent)
}
