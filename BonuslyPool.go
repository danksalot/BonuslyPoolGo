package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"

	"github.com/gocarina/gocsv"
)

type Bonus struct {
	ReceiverEmail string `json:"receiver_email"`
	Amount        int    `json:"amount"`
	Hashtag       string `json:"hashtag"`
	Reason        string `json:"reason"`
}

type Participant struct {
	Username string `csv:"username"`
	Email    string `csv:"email"`
	Token    string `csv:"token"`
	Balance  int    `csv:"giving_balance"`
}

type ResultType struct {
	Email   string `json:"email"`
	Balance int    `json:"giving_balance"`
}

type Response struct {
	Result ResultType `json:"result"`
}

func main() {
	in, err := os.Open("users.csv")
	if err != nil {
		panic(err)
	}
	defer in.Close()

	participants := []*Participant{}

	client := &http.Client{}

	if err := gocsv.UnmarshalFile(in, &participants); err != nil {
		panic(err)
	}

	for _, participant := range participants {
		resp, err := client.Get("https://bonus.ly/api/v1/users/me?access_token=" + participant.Token)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var response Response
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			if err := json.Unmarshal(body, &response); err != nil {
				panic(err)
			}
			participant.Email = response.Result.Email
			participant.Balance = response.Result.Balance
			fmt.Println("Got info for", participant.Email, "- Balance:", participant.Balance)
		} else {
			fmt.Println("Get failed with error: ", resp.Status)
		}
	}

	for _, participant := range participants {
		fmt.Println("Participant:", participant.Username)
		amount := math.Floor(float64(participant.Balance) / float64(len(participants)-1))
		for _, recipient := range participants {
			if participant.Username == recipient.Username || len(participants) == 1 || amount == 0 {
				continue
			}

			bonus := Bonus{
				ReceiverEmail: recipient.Email,
				Amount:        int(amount),
				Hashtag:       "#connection",
				Reason:        "for contributing to the group",
			}

			body, _ := json.Marshal(bonus)
			buffer := bytes.NewBuffer(body)

			req, err := http.NewRequest("POST", "https://bonus.ly/api/v1/bonuses", buffer)
			if err != nil {
				panic(err)
			}

			req.Header.Add("Authorization", "Bearer "+participant.Token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			if resp.StatusCode != http.StatusOK {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}

				panic("Response: " + string(body))
			}

			defer resp.Body.Close()
			fmt.Println("Recipient", recipient.Username, "received", amount, "from", participant.Username)
		}
	}
}
