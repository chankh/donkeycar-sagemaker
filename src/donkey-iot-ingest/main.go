package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

const meta = "{\"types\": [\"image_array\", \"float\", \"float\", \"str\"], \"inputs\": [\"cam/image_array\", \"user/angle\", \"user/throttle\", \"user/mode\"]}"

var (
	// Name of the S3 bucket to store data
	targetBucket = os.Getenv("TARGET_BUCKET")

	// AWS Session for uploading split files into a different AWS Region
	uploadSession = session.Must(session.NewSession())

	// S3 Uploader
	uploader = s3manager.NewUploader(uploadSession)

	t   = time.Now()
	tub = fmt.Sprintf("tub_%d-%d-%d_%d%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())
)

func handler(ctx context.Context, request events.DynamoDBEvent) {
	log.SetOutput(os.Stdout)

	for _, r := range request.Records {
		// we only care about new records
		if r.EventName == "INSERT" {
			newImage := r.Change.NewImage
			currentIX, _ := newImage["current_ix"].Integer()
			userAngle, _ := newImage["user/angle"].Float()
			userThrottle, _ := newImage["user/throttle"].Float()

			t := telemetry{
				VehicleID:     newImage["vehicleID"].String(),
				Time:          newImage["time"].String(),
				CamImageArray: newImage["cam/image_array"].String(),
				CurrentIX:     currentIX,
				Image:         newImage["image"].String(),
				UserAngle:     userAngle,
				UserMode:      newImage["user/mode"].String(),
				UserThrottle:  userThrottle,
			}
			imageBytes, err := base64.StdEncoding.DecodeString(t.Image)
			if err != nil {
				log.Errorln("Cannot decode base64 image", err)
			} else {
				// write image to S3
				imageName := fmt.Sprintf("%s/%s", tub, t.CamImageArray)
				result, err := uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(targetBucket),
					Key:    aws.String(imageName),
					Body:   bytes.NewReader(imageBytes),
				})
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"Bucket": targetBucket,
						"Key":    imageName,
					}).Panic("Unable to upload image to S3")
					return
				}
				log.Infof("Written image to %s/%s", targetBucket, imageName)
				// write json file to S3
				jsonName := fmt.Sprintf("%s/record_%d.json", tub, t.CurrentIX)
				j := jsonRecord{
					UserMode:      t.UserMode,
					CamImageArray: t.CamImageArray,
					UserThrottle:  t.UserThrottle,
					UserAngle:     t.UserAngle,
				}
				log.Infof("json: %+v", j)
				rawJson, err := json.Marshal(j)
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"Bucket": targetBucket,
						"Key":    jsonName,
					}).Panic("Unable to create json record")
				} else {
					result, err = uploader.Upload(&s3manager.UploadInput{
						Bucket: aws.String(targetBucket),
						Key:    aws.String(jsonName),
						Body:   bytes.NewReader(rawJson),
					})
					if err != nil {
						log.WithError(err).WithFields(log.Fields{
							"Bucket": targetBucket,
							"Key":    jsonName,
						}).Panic("Unable to upload json record")
					}
					log.Info(result)
				}
			}
		}
	}
}

func generateMeta() {
	key := fmt.Sprintf("%s/meta.json", tub)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(targetBucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(meta)),
	})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"Bucket": targetBucket,
			"Key":    key,
		}).Panic("Unable to create metadata record")
	}
}

func main() {
	generateMeta()
	lambda.Start(handler)
}

type telemetry struct {
	Time          string  `json:"time"`
	CurrentIX     int64   `json:"current_ix"`
	CamImageArray string  `json:"cam/image_array"`
	Image         string  `json:"image"`
	VehicleID     string  `json:"vehicleID"`
	UserAngle     float64 `json:"user/angle"`
	UserThrottle  float64 `json:"user/throttle"`
	UserMode      string  `json:"user/mode"`
}

type jsonRecord struct {
	UserMode      string  `json:"user/mode"`
	CamImageArray string  `json:"cam/image_array"`
	UserThrottle  float64 `json:"user/throttle"`
	UserAngle     float64 `json:"user/angle"`
}
