package tlapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SearchRequest is a search request.
type SearchRequest struct {
	Categories []int
	Facets     map[string]string
	Query      []string
	Added      string
	OrderBy    string
	Order      string
	Page       int

	res *SearchResponse
	i   int
	p   int
	d   time.Duration
	err error
	mu  sync.Mutex
}

// Search creates a search request.
func Search(query ...string) *SearchRequest {
	return &SearchRequest{
		Query: query,
		Page:  1,
		p:     -1,
		i:     -1,
		d:     5 * time.Second,
	}
}

// WithCategories adds search category filters.
func (req *SearchRequest) WithCategories(categories ...int) *SearchRequest {
	req.Categories = categories
	return req
}

// WithFacets adds search facet filters as string pairs (name, value...).
func (req *SearchRequest) WithFacets(facets ...string) *SearchRequest {
	if len(facets)%2 != 0 {
		panic("facets must be a multiple of 2")
	}
	if req.Facets == nil {
		req.Facets = make(map[string]string)
	}
	for i := 0; i < len(facets); i += 2 {
		req.Facets[facets[i]] = facets[i+1]
	}
	return req
}

// WithFacet adds a single search facet name filter, joining values with a ','.
func (req *SearchRequest) WithFacet(name string, values ...string) *SearchRequest {
	if req.Facets == nil {
		req.Facets = make(map[string]string)
	}
	req.Facets[name] = strings.Join(values, ",")
	return req
}

// WithPage sets the search page filter.
func (req *SearchRequest) WithPage(page int) *SearchRequest {
	req.Page = page
	return req
}

// WithAdded sets the search added filter.
func (req *SearchRequest) WithAdded(added string) *SearchRequest {
	req.Added = added
	return req
}

// WithOrderBy sets the search orderBy parameter (see OrderBy constants).
func (req *SearchRequest) WithOrderBy(orderBy string) *SearchRequest {
	req.OrderBy = orderBy
	return req
}

// WithOrder sets the search order parameter (see Order constants).
func (req *SearchRequest) WithOrder(order string) *SearchRequest {
	req.Order = order
	return req
}

// WithNextDelay sets the next delay, for use if user class is rate limited.
func (req *SearchRequest) WithNextDelay(d time.Duration) *SearchRequest {
	req.d = d
	return req
}

// Do executes the request against the client.
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
	if req.Added != "" {
		q += "/added/" + req.Added
	}
	if req.OrderBy != "" {
		q += "/orderby/" + req.OrderBy
	}
	if req.Order != "" {
		q += "/order/" + req.Order
	}
	if req.Page != 0 {
		q += "/page/" + strconv.Itoa(req.Page)
	}
	httpReq, err := http.NewRequest("GET", "https://www.torrentleech.org/torrents/browse/list"+q, nil)
	if err != nil {
		return nil, err
	}
	res := new(SearchResponse)
	if err := cl.Do(ctx, httpReq, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Next returns true if there are search results available for the request.
//
// Example:
//
//	req := tlapi.Search()
//	for req.Next(ctx, cl) {
//		torrent := req.NextTorrent()
//		/* ... */
//	}
//	if err := req.Err(); err != nil {
//		/* ... */
//	}
func (req *SearchRequest) Next(ctx context.Context, cl *Client) bool {
	req.mu.Lock()
	defer req.mu.Unlock()
	page := req.Page
	if page == 0 {
		page = 1
	}
	switch {
	case req.err != nil:
		return false
	case req.res != nil:
		switch {
		case req.i < len(req.res.TorrentList)-1:
			req.i++
			return true
		case (page+req.p)*req.res.PerPage >= req.res.NumFound:
			return false
		}
	}
	req.p, req.i = req.p+1, 0
	if req.d != 0 && req.p != 0 {
		<-time.After(req.d)
	}
	req.res, req.err = req.WithPage(page+req.p).Do(ctx, cl)
	return req.err == nil && req.i < len(req.res.TorrentList)
}

// Cur returns the search response cursor's current torrent. Returns the same
// value until Next is called. Panics if called prior to Next.
//
// See Next for an overview of using this method.
func (req *SearchRequest) Cur() Torrent {
	req.mu.Lock()
	defer req.mu.Unlock()
	return req.res.TorrentList[req.i]
}

// Err returns the last error in the search response.
//
// See Next for an overview of using this method.
func (req *SearchRequest) Err() error {
	req.mu.Lock()
	defer req.mu.Unlock()
	return req.err
}

// All returns all results for the search request.
func (req *SearchRequest) All(ctx context.Context, cl *Client) ([]Torrent, error) {
	var torrents []Torrent
	for req.Next(ctx, cl) {
		torrents = append(torrents, req.Cur())
	}
	if err := req.Err(); err != nil {
		return nil, err
	}
	return torrents, nil
}

// SearchResponse is a search response.
type SearchResponse struct {
	Facets struct {
		CategoryID FacetID `json:"categoryID"`
		Added      Facet   `json:"added"`
		Name       Facet   `json:"name"`
		Seeders    Facet   `json:"seeders"`
		Size       Facet   `json:"size"`
		Tags       Tags    `json:"tags"`
	} `json:"facets"`
	Facetswoc      map[string]Tags `json:"facetswoc,omitempty"`
	LastBrowseTime Time            `json:"lastBrowseTime"`
	NumFound       int             `json:"numFound,omitempty"`
	OrderBy        string          `json:"orderBy,omitempty"`
	Order          string          `json:"order,omitempty"`
	Page           int             `json:"page,omitempty"`
	PerPage        int             `json:"perPage,omitempty"`
	TorrentList    []Torrent       `json:"torrentList,omitempty"`
	UserTimeZone   string          `json:"userTimeZone,omitempty"`
}

// Facet is a facet.
type Facet struct {
	Items map[string]Item `json:"items,omitempty"`
	Name  string          `json:"name,omitempty"`
	Title string          `json:"title,omitempty"`
	Type  string          `json:"type,omitempty"`
}

// FacetID is a facet id.
type FacetID struct {
	Items map[string]int `json:"items,omitempty"`
	Name  string         `json:"name,omitempty"`
	Title string         `json:"title,omitempty"`
	Type  string         `json:"type,omitempty"`
}

// Item is search facet item.
type Item struct {
	Label string `json:"label,omitempty"`
	Count int    `json:"count,omitempty"`
}

// Tags are tags.
type Tags struct {
	Items map[string]int `json:"items,omitempty"`
	Name  string         `json:"name,omitempty"`
	Title string         `json:"title,omitempty"`
	Type  string         `json:"type,omitempty"`
}

// Torrent is a torrent.
type Torrent struct {
	AddedTimestamp     time.Time `json:"addedTimestamp"`
	CategoryID         int       `json:"categoryID,omitempty"`
	Completed          int       `json:"completed,omitempty"`
	DownloadMultiplier int       `json:"download_multiplier,omitempty"`
	ID                 int       `json:"id,omitempty"`
	Filename           string    `json:"filename,omitempty"`
	Genres             []string  `json:"genres,omitempty"`
	IgdbID             string    `json:"igdbID,omitempty"`
	AnimeID            string    `json:"animeID,omitempty"`
	ImdbID             string    `json:"imdbID,omitempty"`
	Leechers           int       `json:"leechers,omitempty"`
	Name               string    `json:"name,omitempty"`
	New                bool      `json:"new,omitempty"`
	NumComments        int       `json:"numComments,omitempty"`
	Rating             float64   `json:"rating,omitempty"`
	Seeders            int       `json:"seeders,omitempty"`
	Size               int64     `json:"size,omitempty"`
	Tags               []string  `json:"tags,omitempty"`
	TvmazeID           string    `json:"tvmazeID,omitempty"`
	Uploader           string    `json:"uploader,omitempty"`
	CommentsDisabled   int       `json:"commentsDisabled,omitempty"`
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (t *Torrent) UnmarshalJSON(buf []byte) error {
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return err
	}
	var torrent Torrent
	for k, v := range m {
		switch k {
		case "addedTimestamp":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("invalid addedTimestamp type %T", v)
			}
			var err error
			if torrent.AddedTimestamp, err = time.Parse(timefmt, s); err != nil {
				return fmt.Errorf("invalid addedTimestamp value %q: %w", s, err)
			}
		case "categoryID":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid categoryID type %T", v)
			}
			torrent.CategoryID = int(f)
		case "completed":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid completed type %T", v)
			}
			torrent.Completed = int(f)
		case "download_multiplier":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid download_multiplier type %T", v)
			}
			torrent.DownloadMultiplier = int(f)
		case "commentsDisabled":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid commentsDisabled type %T", v)
			}
			torrent.CommentsDisabled = int(f)
		case "fid":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("invalid fid type %T", v)
			}
			var err error
			if torrent.ID, err = strconv.Atoi(s); err != nil {
				return fmt.Errorf("invalid fid value %q: %w", s, err)
			}
		case "filename":
			var ok bool
			if torrent.Filename, ok = v.(string); !ok {
				return fmt.Errorf("invalid filename type %T", v)
			}
		case "genres":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("invalid genres type %T", v)
			}
			torrent.Genres = strings.Split(s, ", ")
		case "igdbID":
			switch x := v.(type) {
			case string:
				torrent.IgdbID = x
			case float64:
				torrent.IgdbID = strconv.Itoa(int(x))
			default:
				return fmt.Errorf("invalid igdbID type %T", v)
			}
		case "animeID":
			switch x := v.(type) {
			case string:
				torrent.AnimeID = x
			case float64:
				torrent.AnimeID = strconv.Itoa(int(x))
			default:
				return fmt.Errorf("invalid animeID type %T", v)
			}
		case "imdbID":
			var ok bool
			if torrent.ImdbID, ok = v.(string); !ok {
				return fmt.Errorf("invalid imdbID type %T", v)
			}
		case "leechers":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid leechers type %T", v)
			}
			torrent.Leechers = int(f)
		case "name":
			var ok bool
			if torrent.Name, ok = v.(string); !ok {
				return fmt.Errorf("invalid name type %T", v)
			}
		case "new":
			var ok bool
			if torrent.New, ok = v.(bool); !ok {
				return fmt.Errorf("invalid new type %T", v)
			}
		case "numComments":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid numComments type %T", v)
			}
			torrent.NumComments = int(f)
		case "rating":
			var ok bool
			if torrent.Rating, ok = v.(float64); !ok {
				return fmt.Errorf("invalid rating type %T", v)
			}
		case "seeders":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid seeders type %T", v)
			}
			torrent.Seeders = int(f)
		case "size":
			f, ok := v.(float64)
			if !ok {
				return fmt.Errorf("invalid size type %T", v)
			}
			torrent.Size = int64(f)
		case "tags":
			switch x := v.(type) {
			case string:
			case []any:
				for i, z := range x {
					s, ok := z.(string)
					if !ok {
						return fmt.Errorf("invalid tags value type %T (pos %d)", z, i)
					}
					torrent.Tags = append(torrent.Tags, s)
				}
			default:
				return fmt.Errorf("invalid tags type %T", v)
			}
		case "tvmazeID":
			var ok bool
			if torrent.TvmazeID, ok = v.(string); !ok {
				return fmt.Errorf("invalid tvmazeID type %T", v)
			}
		case "uploader":
			var ok bool
			if torrent.Uploader, ok = v.(string); !ok {
				return fmt.Errorf("invalid uploader type %T", v)
			}
		default:
			return fmt.Errorf("unknown field %q", k)
		}
	}
	*t = torrent
	return nil
}

// Time is a time value.
type Time struct {
	time.Time
}

// String satisfies the fmt.Stringer interface.
func (t Time) String() string {
	return t.Format(timefmt)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(buf []byte) error {
	if string(buf) == `""` {
		return nil
	}
	i, err := strconv.ParseInt(string(buf[1:len(buf)-1]), 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.Unix(i, 0)
	return nil
}

// Facet filter names.
const (
	FacetAdded   = "added"
	FacetName    = "name"
	FacetSeeders = "seeders"
	FacetSize    = "size"
	FacetTags    = "tags"
)

// Order values.
const (
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

// OrderBy values.
const (
	OrderByNameSort    = "nameSort"
	OrderByAdded       = "added"
	OrderByNumComments = "numComments"
	OrderBySize        = "size"
	OrderByCompleted   = "completed"
	OrderBySeeders     = "seeders"
	OrderByLeechers    = "leechers"
)

// Facet filter values.
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

// Categories.
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

// escaper escapes special characters in facet filters.
var escaper = strings.NewReplacer(
	"[", "%255B",
	" ", "%2520",
	"]", "%255D",
)

// timefmt is the time format used for parsing and displaying time values.
const timefmt = "2006-01-02 15:04:05"
