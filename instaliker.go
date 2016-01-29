package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"runtime"
	"strings"
)

const INSTAGRAM_HOST string = "https://www.instagram.com"
const AUTH_URI string = "/accounts/login/ajax/"
const LIKE_URI string = "/web/likes/%s/like/"

type AuthJson struct {
	Status        string `json:"status"`
	Authenticated bool   `json:"authenticated"`
}

type LikeJson struct {
	Status string `json:"status"`
}

type InstaFeed struct {
	EntryData struct {
		Feedpage []struct {
			Feed struct {
				Media struct {
					Nodes []struct {
						Date    float64 `json:"date"`
						Caption string  `json:"caption"`
						Likes   struct {
							ViewerHasLiked bool `json:"viewer_has_liked"`
						} `json:"likes"`
						Owner struct {
							Username string `json:"username"`
						} `json:"owner"`
						ID         string `json:"id"`
						DisplaySrc string `json:"display_src"`
					} `json:"nodes"`
				} `json:"media"`
			} `json:"feed"`
			Suggesteduserslist interface{} `json:"suggestedUsersList"`
		} `json:"FeedPage"`
	} `json:"entry_data"`
}

func loader(client *http.Client, urlAddr string, header http.Header, data *url.Values) []byte {
	method := "GET"
	if data != nil {
		method = "POST"
	} else {
		data = &url.Values{}
	}
	request, _ := http.NewRequest(method, urlAddr, bytes.NewBufferString(data.Encode()))
	request.Header = header
	resp, _ := client.Do(request)
	defer resp.Body.Close()
	content, _ := ioutil.ReadAll(resp.Body)
	return content
}

func csrfTokenFromCookies(cookiejar *cookiejar.Jar) string {
	urlParse, _ := url.Parse(INSTAGRAM_HOST)
	for _, cookie := range cookiejar.Cookies(urlParse) {
		if cookie.Name == "csrftoken" {
			return cookie.Value
		}
	}
	return ""
}

func instaliker(login *string, password *string, users map[string]bool, custom_users bool) {
	var header = http.Header{
		"x-requested-with": []string{"XMLHttpRequest"},
		"x-instagram-ajax": []string{"1"},
		"user-agent":       []string{"Mozilla/5.0"},
		"authority":        []string{"www.instagram.com"},
		"referer":          []string{"https://www.instagram.com/"},
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	loader(client, INSTAGRAM_HOST, header, nil)
	header.Set("x-csrftoken", csrfTokenFromCookies(jar))
	data := &url.Values{}
	data.Add("username", *login)
	data.Add("password", *password)
	var auth_resp *AuthJson
	json.Unmarshal(loader(client, INSTAGRAM_HOST+AUTH_URI, header, data), &auth_resp)
	if !auth_resp.Authenticated {
		fmt.Println("Access denied!")
		return
	}
	header.Set("x-csrftoken", csrfTokenFromCookies(jar))
	re := regexp.MustCompile("window\\._sharedData = (.+);</script>")
	data_str := re.FindSubmatch(loader(client, INSTAGRAM_HOST, header, data))
	if len(data_str) < 2 {
		fmt.Println("Data parse error!")
		return
	}
	var insta_feed *InstaFeed
	json.Unmarshal(data_str[1], &insta_feed)
	header.Set("x-csrftoken", csrfTokenFromCookies(jar))
	for _, feed := range insta_feed.EntryData.Feedpage {
		done_ch := make(chan bool, len(feed.Feed.Media.Nodes))
		length := 0
		for _, node := range feed.Feed.Media.Nodes {
			like_it, ok := users[node.Owner.Username]
			if custom_users {
				like_it = ok && like_it
			} else {
				like_it = !ok || like_it
			}
			if !node.Likes.ViewerHasLiked && like_it {
				fmt.Println(fmt.Sprintf("Like user %s, Image: \"%s\" URL: %s", node.Owner.Username,
					node.Caption,
					node.DisplaySrc))
				length = length + 1
				go func(done chan<- bool, urlLike string) {
					loader(client, urlLike, header, &url.Values{})
					done <- true
				}(done_ch, fmt.Sprintf(INSTAGRAM_HOST+LIKE_URI, node.ID))
			}
		}
		for i := 0; i < length; i = i + 1 {
			<-done_ch
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	login := flag.String("login", "", "instagram login")
	password := flag.String("password", "", "instagram password")
	exclude_users_string := flag.String("exclude", "", "exclude from autolike (comma-separated list)")
	custom_users_string := flag.String("users", "", "autolike only for custom users (comma-separated list)")
	flag.Parse()
	if *login == "" {
		fmt.Printf("Instagram login not found")
		return
	}
	if *password == "" {
		fmt.Printf("Instagram password not found")
		return
	}
	users := make(map[string]bool)
	if (*exclude_users_string != "") && (*custom_users_string != "") {
		fmt.Printf("Use exclude or users")
		return
	}

	for _, excl_user := range strings.Split(*exclude_users_string, ",") {
		if excl_user != "" {
			users[excl_user] = false
		}
	}
	for _, incl_user := range strings.Split(*custom_users_string, ",") {
		if incl_user != "" {
			users[incl_user] = true
		}
	}
	instaliker(login, password, users, *custom_users_string != "")
}
