package saavn

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const (
	BASE_URL = "https://www.jiosaavn.com/api.php"
	VERSION  = "4"
)

type Saavn struct {
	client *http.Client
}

type Playlist struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Songs []Song `json:"list"`
}

type Song struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Info  Info   `json:"more_info"`
}

type Info struct {
	EncryptedMediaURL string `json:"encrypted_media_url"`
}

type AuthResponse struct {
	AuthURL string `json:"auth_url"`
	Type    string `json:"type"`
	Status  string `json:"status"`
}

func New() *Saavn {
	return &Saavn{
		client: &http.Client{},
	}
}

func (s *Saavn) makeRequest(method string, url string, body io.Reader, params *url.Values) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Attach query params
	req.URL.RawQuery = params.Encode()

	// Add headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("accept", "application/json")

	return req, nil
}

// Get playlist
func (s *Saavn) GetPlaylist(playlistURL string) (Playlist, error) {
	params := &url.Values{}
	params.Add("__call", "webapi.get")
	params.Add("token", playlistURL)
	params.Add("type", "playlist")
	params.Add("p", "1")
	params.Add("n", "100")
	params.Add("api_version", VERSION)
	params.Add("_format", "json")
	params.Add("_marker", "0")

	p := Playlist{}

	req, err := s.makeRequest(http.MethodGet, BASE_URL, nil, params)
	if err != nil {
		return p, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&p)
	if err != nil {
		return p, err
	}

	return p, nil
}

// Get song download url
func (s *Saavn) GetSongDownloadURL(song Song) (string, error) {
	params := &url.Values{}
	params.Add("__call", "song.generateAuthToken")
	params.Add("url", song.Info.EncryptedMediaURL)
	params.Add("bitrate", "128")
	params.Add("api_version", VERSION)
	params.Add("_format", "json")
	params.Add("_marker", "0")

	req, err := s.makeRequest(http.MethodGet, BASE_URL, nil, params)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	r := AuthResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return "", err
	}

	return r.AuthURL, nil
}
