package auto

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/segmentio/ksuid"
)

type Vehicle struct {
	ID                       string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	MetersTraveled           int64
	MetersTraveledType       string
	LastKnownUserInputMeters int64

	// Automatic Attributes
	AutomaticID        string
	FuelGrade          string
	AutomaticCreatedAt time.Time
	AutomaticUpdatedAt time.Time
	Make               string
	Model              string
	SubModel           string
	Year               int
}

// UpdateVehiclesForAccount runs the update for vehicles belonging to that account.
func UpdateVehiclesForAccount(account *Account, token *AutomaticAccessToken) error {
	uri := AutomaticAPIURL("/vehicle")
	uri.RawQuery = "limit=50"
	request, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}

	token.SignRequest(request)

	response, err := SendRequest(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	type responseVehicle struct {
		AutomaticID        string    `json:"id"`
		FuelGrade          string    `json:"fuel_grade"`
		AutomaticCreatedAt time.Time `json:"created_at"`
		AutomaticUpdatedAt time.Time `json:"updated_at"`
		Make               string    `json:"make"`
		Model              string    `json:"model"`
		SubModel           string    `json:"submodel"`
		Year               int       `json:"year"`
	}

	type responseObject struct {
		Vehicles []responseVehicle `json:"results"`
	}

	object := responseObject{}

	err = json.Unmarshal(body, &object)
	if err != nil {
		return err
	}

	write := func(items []*dynamodb.TransactWriteItem) error {
		if len(items) == 0 {
			return nil
		}

		_, err := DynamoDB().TransactWriteItems(&dynamodb.TransactWriteItemsInput{TransactItems: items})

		return err
	}

	items := []*dynamodb.TransactWriteItem{}
	for _, result := range object.Vehicles {
		id := ksuid.New().String()
		currentTime := time.Now()

		pk := fmt.Sprintf("vehicle/%s", result.AutomaticID)
		sk := fmt.Sprintf("_VEHICLE_")

		expressions := []string{
			"#id = if_not_exists(#id, :id)",
			"#created_at = if_not_exists(#created_at, :created_at)",
			"#updated_at = :updated_at",
			"#meters_traveled = if_not_exists(#meters_traveled, :meters_traveled)",
			"#meters_traveled_type = if_not_exists(#meters_traveled_type, :meters_traveled_type)",
			"#last_user_input_meters = if_not_exists(#last_user_input_meters, :last_user_input_meters)",
			"GSI2PK = if_not_exists(GSI2PK, :id)",
			"GSI2SK = if_not_exists(GSI2SK, :gsi2sk)",
			"#last_updated_for = :last_updated_for",
			"#automatic_id = :automatic_id",
			"#fuel_grade = :fuel_grade",
			"#a_created_at = :a_created_at",
			"#a_updated_at = :a_updated_at",
			"#make = :make",
			"#model = :model",
			"#submodel = :submodel",
			"#year = :year",
		}

		update := &dynamodb.Update{
			TableName: TableName(),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {S: aws.String(pk)},
				"SK": {S: aws.String(sk)},
			},
			UpdateExpression: aws.String(fmt.Sprintf("SET %s", strings.Join(expressions, ", "))),
			ExpressionAttributeNames: map[string]*string{
				"#id":                     aws.String("VehicleID"),
				"#created_at":             aws.String("CreatedAt"),
				"#updated_at":             aws.String("UpdatedAt"),
				"#meters_traveled":        aws.String("MetersTraveled"),
				"#meters_traveled_type":   aws.String("MetersTraveledType"),
				"#last_user_input_meters": aws.String("LastKnownUserInputMeters"),
				"#automatic_id":           aws.String("A_ID"),
				"#fuel_grade":             aws.String("A_FuelGrade"),
				"#a_created_at":           aws.String("A_CreatedAt"),
				"#a_updated_at":           aws.String("A_UpdatedAt"),
				"#make":                   aws.String("A_Make"),
				"#model":                  aws.String("A_Model"),
				"#submodel":               aws.String("A_SubModel"),
				"#year":                   aws.String("A_Year"),
				"#last_updated_for":       aws.String("M_LastUpdatedFor"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":id":                     {S: aws.String(id)},
				":created_at":             DynamoTime(currentTime),
				":updated_at":             DynamoTime(currentTime),
				":meters_traveled":        {N: aws.String("0")},
				":meters_traveled_type":   {S: aws.String("best_guess")},
				":automatic_id":           {S: aws.String(result.AutomaticID)},
				":fuel_grade":             {S: aws.String(result.FuelGrade)},
				":a_created_at":           DynamoTime(result.AutomaticCreatedAt),
				":a_updated_at":           DynamoTime(result.AutomaticUpdatedAt),
				":make":                   {S: aws.String(result.Make)},
				":model":                  {S: aws.String(result.Model)},
				":submodel":               {S: aws.String(result.SubModel)},
				":year":                   {N: aws.String(fmt.Sprintf("%d", result.Year))},
				":gsi2sk":                 {S: aws.String(fmt.Sprintf("_VEHICLE_/%s", ksuid.New().String()))},
				":last_updated_for":       {S: aws.String(account.ID)},
				":last_user_input_meters": {N: aws.String("0")},
			},
		}

		put := &dynamodb.Put{
			TableName: TableName(),
			Item: map[string]*dynamodb.AttributeValue{
				"PK":              {S: aws.String(fmt.Sprintf("va/%s", result.AutomaticID))},
				"SK":              {S: aws.String(fmt.Sprintf("va/%s", account.ID))},
				"GSI1PK":          {S: aws.String(fmt.Sprintf("av/%s", account.ID))},
				"GSI1SK":          {S: aws.String(fmt.Sprintf("av/%s", result.AutomaticID))},
				"AccountID":       {S: aws.String(account.ID)},
				"AutomaticID":     {S: aws.String(result.AutomaticID)},
				"M_LastUpdatedAt": DynamoTime(time.Now()),
				"M_Type":          {S: aws.String("vehicle_account")},
			},
		}

		items = append(items, &dynamodb.TransactWriteItem{Update: update}, &dynamodb.TransactWriteItem{Put: put})

		if len(items) >= 20 {
			write(items)
			items = []*dynamodb.TransactWriteItem{}
		}
	}

	write(items)

	return nil
}

// VehiclesForAccount returns all the vehicles present for an account
func VehiclesForAccount(account *Account) ([]*Vehicle, error) {
	query := &dynamodb.QueryInput{
		TableName:              TableName(),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :sk)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(fmt.Sprintf("av/%s", account.ID))},
			":sk": {S: aws.String("av/")},
		},
		Limit:                aws.Int64(25),
		ProjectionExpression: aws.String("AutomaticID"),
	}

	output, err := DynamoDB().Query(query)
	if err != nil {
		return []*Vehicle{}, err
	}

	if aws.Int64Value(output.Count) == 0 {
		return []*Vehicle{}, nil
	}

	keys := []map[string]*dynamodb.AttributeValue{}
	for _, item := range output.Items {
		vID := aws.StringValue(item["AutomaticID"].S)

		keys = append(keys, map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(fmt.Sprintf("vehicle/%s", vID))},
			"SK": {S: aws.String("_VEHICLE_")},
		})
	}

	table := aws.StringValue(TableName())

	requestItems := map[string]*dynamodb.KeysAndAttributes{}
	requestItems[table] = &dynamodb.KeysAndAttributes{
		Keys: keys,
	}

	results, err := DynamoDB().BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: requestItems,
	})
	if err != nil {
		return []*Vehicle{}, err
	}

	vehicles := []*Vehicle{}

	for _, item := range results.Responses[table] {
		fmt.Println(item)

		// 	ID                       string
		// CreatedAt                time.Time
		// UpdatedAt                time.Time
		// MetersTraveled           int64
		// MetersTraveledType       string
		// LastKnownUserInputMeters int64

		// // Automatic Attributes
		// AutomaticID        string
		// FuelGrade          string
		// AutomaticCreatedAt time.Time
		// AutomaticUpdatedAt time.Time
		// Make               string
		// Model              string
		// SubModel           string
		// Year               int
	}

	return vehicles, nil
}
