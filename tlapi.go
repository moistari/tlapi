package tlapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SearchRequest struct {
	Categories []int
	Facets     map[string]string
	Query      []string
	Page       int
}

func Search(query ...string) *SearchRequest {
	return &SearchRequest{
		Query: query,
	}
}

func (req *SearchRequest) Do(ctx context.Context, cl *Client) (*SearchResponse, error) {
	var q string
	if len(req.Categories) != 0 {
		var v []string
		for _, c := range req.Categories {
			v = append(v, strconv.Itoa(c))
		}
		q += "/categories/" + strings.Join(v, ",")
	}
	if req.Facets != nil {
		var v []string
		for _, key := range []string{"added", "name", "seeders", "size", "tags"} {
			if s, ok := req.Facets[key]; ok {
				v = append(v, key+"%3A"+escaper.Replace(s))
			}
		}
		q += "/facets/" + strings.Join(v, "_")
	}
	if len(req.Query) != 0 {
		q += "/query/" + url.PathEscape(strings.Join(req.Query, " "))
	}
	if req.Page != 0 {
		q += "/page/" + strconv.Itoa(req.Page)
	}
	urlstr := "https://www.torrentleech.org/torrents/browse/list" + q
	httpReq, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return nil, err
	}
	res := new(SearchResponse)
	if err := cl.Do(ctx, httpReq, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (req SearchRequest) WithCategories(categories ...int) *SearchRequest {
	req.Categories = categories
	return &req
}

func (req SearchRequest) WithFacets(facets ...string) *SearchRequest {
	if len(facets)%2 != 0 {
		panic("facets must be a multiple of 2")
	}
	if req.Facets == nil {
		req.Facets = make(map[string]string)
	}
	for i := 0; i < len(facets); i += 2 {
		req.Facets[facets[i]] = facets[i+1]
	}
	return &req
}

func (req SearchRequest) WithPage(page int) *SearchRequest {
	req.Page = page
	return &req
}

type SearchResponse struct {
	Facets struct {
		Added   Facet `json:"added,omitempty"`
		Name    Facet `json:"name,omitempty"`
		Seeders Facet `json:"seeders,omitempty"`
		Size    Facet `json:"size,omitempty"`
		Tags    Tags  `json:"tags,omitempty"`
	} `json:"facets,omitempty"`
	Facetswoc      map[string]Tags `json:"facetswoc,omitempty"`
	LastBrowseTime Time            `json:"lastBrowseTime,omitempty"`
	NumFound       int             `json:"numFound,omitempty"`
	OrderBy        string          `json:"orderBy,omitempty"`
	Order          string          `json:"order,omitempty"`
	Page           int             `json:"page,omitempty"`
	PerPage        int             `json:"perPage,omitempty"`
	TorrentList    []Torrent       `json:"torrentList,omitempty"`
	UserTimeZone   string          `json:"userTimeZone,omitempty"`
}

type Facet struct {
	Items map[string]Item `json:"items,omitempty"`
	Name  string          `json:"name,omitempty"`
	Title string          `json:"title,omitempty"`
	Type  string          `json:"type,omitempty"`
}

type Item struct {
	Label string `json:"label,omitempty"`
	Count int    `json:"count,omitempty"`
}

type Tags struct {
	Items map[string]int `json:"items,omitempty"`
	Name  string         `json:"name,omitempty"`
	Title string         `json:"title,omitempty"`
	Type  string         `json:"type,omitempty"`
}

type Torrent struct {
	AddedTimestamp     Timestamp `json:"addedTimestamp,omitempty"`
	CategoryID         int       `json:"categoryID,omitempty"`
	Completed          int       `json:"completed,omitempty"`
	DownloadMultiplier int       `json:"download_multiplier,omitempty"`
	FID                string    `json:"fid,omitempty"`
	Filename           string    `json:"filename,omitempty"`
	Genres             Genres    `json:"genres,omitempty"`
	IgdbID             IgdbID    `json:"igdbID,omitempty"`
	ImdbID             string    `json:"imdbID,omitempty"`
	Leechers           int       `json:"leechers,omitempty"`
	Name               string    `json:"name,omitempty"`
	New                bool      `json:"new,omitempty"`
	NumComments        int       `json:"numComments,omitempty"`
	Rating             float64   `json:"rating,omitempty"`
	Seeders            int       `json:"seeders,omitempty"`
	Size               int64     `json:"size,omitempty"`
	Tags               TagList   `json:"tags,omitempty"`
	TvmazeID           string    `json:"tvmazeID,omitempty"`
	Uploader           string    `json:"uploader,omitempty"`
}

type Time time.Time

func (t Time) String() string {
	return time.Time(t).Format(timestampfmt)
}

func (t *Time) UnmarshalJSON(buf []byte) error {
	i, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		return err
	}
	*t = Time(time.Unix(i, 0))
	return nil
}

type Genres []string

func (g *Genres) UnmarshalJSON(buf []byte) error {
	if len(buf) < 2 {
		return errors.New("invalid genres value")
	}
	*g = strings.Split(string(buf[1:len(buf)-1]), ", ")
	return nil
}

type TagList []string

func (l *TagList) UnmarshalJSON(buf []byte) error {
	if string(buf) == `""` {
		return nil
	}
	var v []string
	if err := json.Unmarshal(buf, &v); err != nil {
		return err
	}
	*l = TagList(v)
	return nil
}

type IgdbID string

func (i *IgdbID) UnmarshalJSON(buf []byte) error {
	if string(buf) == `""` {
		return nil
	}
	*i = IgdbID(string(buf))
	return nil
}

type Timestamp time.Time

func (t Timestamp) String() string {
	return time.Time(t).Format(timestampfmt)
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte("\"" + t.String() + "\""), nil
}

func (t *Timestamp) UnmarshalJSON(buf []byte) error {
	if len(buf) < 2 {
		return errors.New("invalid timestamp value")
	}
	v, err := time.Parse(timestampfmt, string(buf[1:len(buf)-1]))
	if err != nil {
		return err
	}
	*t = Timestamp(v)
	return nil
}

const timestampfmt = "2006-01-02 15:04:05"

const (
	RangeLast2Weeks  = "[NOW/HOUR-14DAYS TO NOW/HOUR+1HOUR]"
	RangeLastMonth   = "[NOW/HOUR-1MONTH TO NOW/HOUR+1HOUR]"
	RangeLastWeek    = "[NOW/HOUR-7DAYS TO NOW/HOUR+1HOUR]"
	RangeLast24Hours = "[NOW/MINUTE-24HOURS TO NOW/MINUTE+1MINUTE]"
	RangeLast48Hours = "[NOW/MINUTE-48HOURS TO NOW/MINUTE+1MINUTE]"
	RangeLast72Hours = "[NOW/MINUTE-72HOURS TO NOW/MINUTE+1MINUTE]"
	Seeders0to50     = "[0 TO 50]"
	Seeders200Plus   = "[201 TO *]"
	Seeders50to200   = "[51 TO 200]"
	Size0to750MB     = "[0 TO 786432000]"
	Size1_5GBto4_5GB = "[1610612736 TO 4831838208]"
	Size15GBPlus     = "[16106127360 TO *]"
	Size4_5GBto15GB  = "[4831838208 TO 16106127360]"
	Size750MBto1_5GB = "[786432000 TO 1610612736]"
)

var escaper = strings.NewReplacer(
	"[", "%255B",
	" ", "%2520",
	"]", "%255D",
)

const (
	CategoryMoviesCam               = 8
	CategoryMoviesTSTC              = 9
	CategoryMoviesDVDRipDVDScreener = 11
	CategoryMoviesWebRip            = 37
	CategoryMoviesHDRip             = 43
	CategoryMoviesBluRayRip         = 14
	CategoryMoviesDVDR              = 12
	CategoryMoviesBluRay            = 13
	CategoryMovies4k                = 47
	CategoryMoviesBoxsets           = 15
	CategoryMoviesDocumentaries     = 29

	CategoryTVEpisodes   = 26
	CategoryTVEpisodesHD = 32
	CategoryTVBoxsets    = 27

	CategoryGamesPC             = 17
	CategoryGamesMac            = 42
	CategoryGamesXbox           = 18
	CategoryGamesXbox360        = 19
	CategoryGamesXboxOne        = 40
	CategoryGamesPS2            = 20
	CategoryGamesPS3            = 21
	CategoryGamesPS4            = 39
	CategoryGamesPS5            = 49
	CategoryGamesPSP            = 22
	CategoryGamesWii            = 28
	CategoryGamesNintendoDS     = 30
	CategoryGamesNintendoSwitch = 48

	CategoryAppsPCISO  = 23
	CategoryAppsMac    = 24
	CategoryAppsMobile = 25
	CategoryApps0Day   = 33

	CategoryEducation = 38

	CategoryAnimationAnime    = 34
	CategoryAnimationCartoons = 35

	CategoryBooksEbooks = 45
	CategoryBooksComics = 46

	CategoryMusicAudio  = 31
	CategoryMusicVideos = 16

	CategoryForeignMovies   = 36
	CategoryForeignTVSeries = 44
)
