package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// requestToken gets the authorization token from reddit
func requestToken(user, pass, agent, public, private string) string {
	client := &http.Client{}
	URL := "https://www.reddit.com/api/v1/access_token"
	// set values for post request body
	vals := url.Values{}
	vals.Set("grant_type", "password")
	vals.Set("username", user)
	vals.Set("password", pass)
	req, err := http.NewRequest("POST", URL, strings.NewReader(vals.Encode()))
	if err != nil {
		fmt.Printf("Failed to create new POST request for %s\n", URL)
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// set user agent and sets Oauth tokens
	req.Header.Add("User-Agent", agent)
	req.SetBasicAuth(public, private)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to get resposnse")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	type redditResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		fmt.Println("Failed to read body")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	res := redditResponse{}
	_ = json.Unmarshal(body, &res)
	token := res.AccessToken + " " + res.TokenType

	fmt.Printf("Response Header: %+v\n", resp.Header)
	fmt.Printf("%s\n", token)
	return token
}

func getSubTop(sub, agent, token, period, limit string) {
	fmt.Printf("Getting %s's top posts of this month\n", sub)
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/top/.json", sub)

	client := &http.Client{}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		fmt.Printf("Failed to create new GET request for %s\n", URL)
		fmt.Println(err.Error())
		os.Exit(1)
	}
	req.Header.Add("Authorization", token)
	req.Header.Add("User-Agent", agent)

	query := req.URL.Query()
	query.Add("t", period)
	query.Add("limit", limit)
	req.URL.RawQuery = query.Encode()

	fmt.Println("---------------------")
	fmt.Printf("URL: %s\n\n", req.URL.String())

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to get resposnse")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		fmt.Println("Failed to read body")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	type postInfo struct {
		Title       string  `json:"title"`
		Sub         string  `json:"subreddit"`
		UpvoteRatio float64 `json:"upvote_ratio"`
		Upvotes     int64   `json:"ups"`
		Permalink   string  `json:"permalink"`
		URL         string  `json:"url"`
		UtcEpoch    float64 `json:"created"`
	}

	type post struct {
		Data postInfo `json:"data"`
	}

	type data struct {
		Children []post `json:"children"`
	}

	type top struct {
		Data data `json:"data"`
	}

	/*
		res := post{}
		_ = json.Unmarshal(body, &res)

		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("Response Header: %+v\n", resp.Header)
		fmt.Println("---------------------")
		fmt.Printf("Response Body: %+v\n", string(body))
	*/
	topRes := top{}
	_ = json.Unmarshal(body, &topRes)

	for _, v := range topRes.Data.Children {
		fmt.Println("---------------------")
		fmt.Printf("%+v\n", v.Data.Title)
		fmt.Printf("%+v\n", v.Data.Upvotes)
		fmt.Printf("%+v\n", v.Data.UpvoteRatio)
		fmt.Printf("%+v\n", v.Data.Permalink)
		fmt.Printf("%+v\n", v.Data.URL)
		fmt.Printf("%+v\n", time.Unix(int64(v.Data.UtcEpoch), 0))
	}
}

func getSubsDB() []string {
	const (
		host   = "localhost"
		port   = 5432
		user   = "nick"
		dbname = "LocalDB"
	)

	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		host, port, user, dbname)

	fmt.Println(dbInfo)

	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully connected to: %s\n", dbname)
	fmt.Println("---------------------")

	sqlStatement := `SELECT name FROM "Subreddits";`
	rows, err := db.Query(sqlStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	subs := make([]string, 0)
	for rows.Next() {
		row := ""
		err = rows.Scan(&row)
		if err != nil {
			panic(err)
		}
		subs = append(subs, row)
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return subs
}

func main() {
	exec := flag.Bool("red", false, "a bool that represents whether or not to use reddit api")
	flag.Parse()

	user := os.Getenv("USER")
	pass := os.Getenv("PASS")
	agent := os.Getenv("AGENT")
	public := os.Getenv("PUBLIC")
	private := os.Getenv("PRIVATE")
	period := "month"
	limit := "1"

	fmt.Println("---------------------")
	fmt.Printf("User: %s\n", user)
	fmt.Printf("Pass: %s\n", pass)
	fmt.Printf("agent: %s\n", agent)
	fmt.Printf("public token: %s\n", public)
	fmt.Printf("private token: %s\n", private)
	fmt.Println("---------------------")

	subs := getSubsDB()
	fmt.Printf("%+v\n", subs)

	if *exec {
		token := requestToken(user, pass, agent, public, private)
		for _, v := range subs {
			fmt.Println("---------------------")
			getSubTop(v, agent, token, period, limit)
			fmt.Println("---------------------")
		}
	}
}
