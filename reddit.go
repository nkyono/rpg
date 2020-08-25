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

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Post represents the item inside of the database that represents a reddit post
type Post struct {
	id          int64
	Title       string
	Upvotes     int64
	UpvoteRatio float64
	URL         string
	Permalink   string
	Date        time.Time
	Sub         string
}

func getDBInstance() *sql.DB {
	const (
		host   = "localhost"
		port   = 5432
		user   = "nick"
		dbname = "LocalDB"
	)

	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		host, port, user, dbname)

	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		panic(err)
	}
	// defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	return db
}

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

	return token
}

func getSubTop(sub, agent, token, period, limit string) {
	// fmt.Printf("Getting %s's top posts of this month\n", sub)
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
		/*
			fmt.Println("---------------------")
			fmt.Printf("%+v\n", v.Data.Title)
			fmt.Printf("%+v\n", v.Data.Upvotes)
			fmt.Printf("%+v\n", v.Data.UpvoteRatio)
			fmt.Printf("%+v\n", v.Data.Permalink)
			fmt.Printf("%+v\n", v.Data.URL)
			fmt.Printf("%+v\n", time.Unix(int64(v.Data.UtcEpoch), 0))
		*/
		item := Post{
			Title:       v.Data.Title,
			Upvotes:     v.Data.Upvotes,
			UpvoteRatio: v.Data.UpvoteRatio,
			Permalink:   v.Data.Permalink,
			URL:         v.Data.URL,
			Date:        time.Unix(int64(v.Data.UtcEpoch), 0),
			Sub:         v.Data.Sub,
		}
		addPost(item)
	}
}

func getSubsDB() []string {
	db := getDBInstance()
	defer db.Close()

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

// Maybe change to take in database instance and pass from other function so only opened and closed once
func addPost(item Post) bool {
	db := getDBInstance()
	defer db.Close()

	sqlStatement := `INSERT INTO "Reddit_Posts" 
					("Title", "Upvotes", "UpvoteRatio", "URL", "Permalink", "Date", "Sub_id")
					VALUES
					($1, $2, $3, $4, $5, $6, (SELECT id FROM "Subreddits" WHERE "name"=$7))`
	_, err := db.Exec(sqlStatement,
		item.Title,
		item.Upvotes,
		item.UpvoteRatio,
		item.URL,
		item.Permalink,
		item.Date,
		item.Sub)

	if err != nil {
		fmt.Printf("Error adding post: %+v", err.Error())
		return false
	}

	return true
}

func deleteAllPosts() bool {
	db := getDBInstance()
	defer db.Close()

	sqlStatement := `TRUNCATE "Reddit_Posts"`
	_, err := db.Exec(sqlStatement)

	if err != nil {
		fmt.Printf("Error emptying table: %+v", err.Error())
		return false
	}

	return true
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading .env file: %+v", err.Error())
		os.Exit(1)
	}
	exec := flag.Bool("red", false, "a bool that represents whether or not to use reddit api")
	flag.Parse()

	user := os.Getenv("USERNAME")
	pass := os.Getenv("PASSWORD")
	agent := os.Getenv("AGENT")
	public := os.Getenv("PUBLIC")
	private := os.Getenv("PRIVATE")
	period := "year"
	limit := "200"

	fmt.Println("---------------------")
	fmt.Printf("User: %s\n", user)
	fmt.Printf("Pass: %s\n", pass)
	fmt.Printf("agent: %s\n", agent)
	fmt.Printf("public token: %s\n", public)
	fmt.Printf("private token: %s\n", private)
	fmt.Println("---------------------")

	subs := getSubsDB()
	fmt.Printf("%+v\n", subs)
	// deleteAllPosts()
	if *exec {
		token := requestToken(user, pass, agent, public, private)
		for _, v := range subs {
			getSubTop(v, agent, token, period, limit)
		}
	}
}
