package auto

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// UpdateTripsForAccount performs a single request to the trips endpoint
func UpdateTripsForAccount(account *Account, token *AutomaticAccessToken, uri *url.URL) (*url.URL, error) {
	request, err := http.NewRequest("GET", uri.String(), nil)
	token.SignRequest(request)

	response, err := SendRequest(request)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	type tripResponseMetadata struct {
		Next string `json:"next"`
	}

	type apiTrip struct {
		ID         string    `json:"id"`
		Duration   float64   `json:"duration_s"`
		Distance   float64   `json:"distance_m"`
		StartedAt  time.Time `json:"started_at"`
		EndedAt    time.Time `json:"ended_at"`
		DriverURL  string    `json:"driver"`
		VehicleURL string    `json:"vehicle"`
	}

	type tripResponse struct {
		Metadata tripResponseMetadata `json:"_metadata"`
		Trips    []apiTrip            `json:"results"`
	}

	inBatches := func(input []apiTrip, fn func([]apiTrip) error) error {
		batch := 25

		for i := 0; i < len(input); i += batch {
			j := i + batch
			if j > len(input) {
				j = len(input)
			}

			err := fn(input[i:j])
			if err != nil {
				return err
			}
		}

		return nil
	}

	results := tripResponse{}

	err = json.Unmarshal(body, &results)
	if err != nil {
		return nil, err
	}

	err = inBatches(results.Trips, func(trips []apiTrip) error {
		ops := []*dynamodb.WriteRequest{}

		for _, trip := range trips {
			vURL, _ := url.Parse(trip.VehicleURL)
			dURL, _ := url.Parse(trip.DriverURL)

			vID := filepath.Base(vURL.Path)
			dID := filepath.Base(dURL.Path)

			ops = append(ops, &dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"PK":        {S: aws.String(fmt.Sprintf("trip/%s", trip.ID))},
						"SK":        {S: aws.String(fmt.Sprintf("vehicle/%s", vID))},
						"GSI1PK":    {S: aws.String(fmt.Sprintf("vehicle/%s", vID))},
						"GSI1SK":    {S: aws.String(fmt.Sprintf("trip/%s", trip.ID))},
						"GSI2PK":    {S: aws.String(fmt.Sprintf("user/%s", dID))},
						"GSI2SK":    {S: aws.String(fmt.Sprintf("trip/%s", trip.ID))},
						"TripID":    {S: aws.String(trip.ID)},
						"DurationS": {N: aws.String(fmt.Sprintf("%0.0f", trip.Duration))},
						"DistanceM": {N: aws.String(fmt.Sprintf("%0.02f", trip.Distance))},
						"StartedAt": DynamoTime(trip.StartedAt),
						"EndedAt":   DynamoTime(trip.EndedAt),
					},
				},
			})
		}

		if len(ops) == 0 {
			return nil
		}

		requests := map[string][]*dynamodb.WriteRequest{}
		requests[aws.StringValue(TableName())] = ops

		_, err := DynamoDB().BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: requests,
		})

		return err
	})
	if err != nil {
		return nil, err
	}

	next, _ := url.Parse(results.Metadata.Next)

	return next, nil
}
