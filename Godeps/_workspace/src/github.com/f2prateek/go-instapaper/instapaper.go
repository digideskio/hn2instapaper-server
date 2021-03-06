package instapaper

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/segmentio/url-encoder"
)

// Client wraps the Instapaper Simple API https://www.instapaper.com/api/simple.
type Client struct {
	httpClient *http.Client
	username   string
	password   string
}

// New returns a new Instapaper client.
func New(username, password string) *Client {
	return &Client{
		httpClient: http.DefaultClient,
		username:   username,
		password:   password,
	}
}

// AddParams represents the parameters that should can be provided when adding
// a URL to a user account.
type AddParams struct {
	URL       string  `url:"url"`
	Title     *string `url:"title"`
	Selection *string `url:"selection"`
}

// AddReponse represents the result of adding a url to instapaper.
type AddReponse struct {
	BookmarkID int `json:"bookmark_id"`
}

// Add will add the given URL and parameters to the user account.
func (c *Client) Add(params AddParams) (AddReponse, error) {
	v := encoder.Marshal(params)
	v.Add("username", c.username)
	v.Add("password", c.password)

	response, err := http.PostForm("https://www.instapaper.com/api/add", v)
	if err != nil {
		return AddReponse{}, err
	}

	var result AddReponse
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return AddReponse{}, err
	}

	return result, nil
}

// Authenticate the client credentials.
func (c *Client) Authenticate() (bool, error) {
	v := url.Values{}
	v.Add("username", c.username)
	v.Add("password", c.password)

	response, err := http.PostForm("https://www.instapaper.com/api/authenticate", v)
	if err != nil {
		return false, err
	}

	if response.StatusCode == 200 {
		return true, nil
	}

	return false, nil
}
