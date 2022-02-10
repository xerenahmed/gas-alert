package main

import (
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

var client *resty.Client
var targetGas float64

func init() {
	var err error
	if err = godotenv.Load(); err != nil {
		logrus.WithError(err).Fatal("Error loading .env file")
	}

	targetGas, err = strconv.ParseFloat(os.Getenv("BELOW_GAS"), 64)
	if err != nil {
		logrus.WithError(err).WithField("gas", os.Getenv("BELOW_GAS")).Fatal("Invalid target gas")
	}

	client = resty.New().SetBaseURL("https://api.etherscan.io/api")

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
}

func main() {
	for {
		time.Sleep(time.Second * 10)

		var res EtherscanResponse
		_, err := client.R().SetQueryParams(map[string]string{
			"module":   "gastracker",
			"action":   "gasoracle",
			"gasprice": "2000000000",
			"apikey":   os.Getenv("ETHERSCAN_API_KEY"),
		}).SetResult(&res).Get("")
		if err != nil {
			logrus.WithError(err).Error("Error getting data from OpenSea")
			continue
		}

		if res.Status != "1" {
			logrus.WithField("status", res.Status).Error("Unexpected status from EtherScan")
			continue
		}

		gas, err := strconv.ParseFloat(res.Result.SafeGasPrice, 64)
		if err != nil {
			logrus.WithError(err).Error("Error parsing gas price")
			continue
		}

		if gas <= targetGas {
			logrus.Infof("Target gas reached: %.4f\n", gas)
			if err := sendMailFloorAlert(gas); err != nil {
				logrus.WithError(err).Error("Error sending mail")
			} else {
				time.Sleep(time.Minute * 5)
			}
		}
	}
}

type EtherscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		SafeGasPrice string `json:"SafeGasPrice"`
	} `json:"result"`
}

func sendMailFloorAlert(gas float64) error {
	from := mail.NewEmail("Gas Alert", os.Getenv("SENDGRID_FROM_EMAIL"))
	subject := "Gas is now " + strconv.FormatFloat(gas, 'f', 1, 64)
	to := mail.NewEmail("Target user", os.Getenv("ALERT_EMAIL"))
	message := mail.NewSingleEmail(from, subject, to, subject, subject)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	_, err := client.Send(message)

	return err
}
