package feedreader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

// Feed represent an RSS/Atom feed
type Feed struct {
	URL   string      `json:"url"`
	Items []*FeedItem `json:"items"`
}

// FeedItem stores info of feed entry
type FeedItem struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	URL       string `json:"url"`
	Published string `json:"published"`
	GUID      string `json:"guid"`
}

type FeedReader struct {
	Sources []string
	Feeds   []Feed
}

// New creates a feedreader
func New(feedsFile string) (FeedReader, error) {
	feedreader := FeedReader{}
	feeds, err := loadFeeds(feedsFile)
	if err != nil {
		return feedreader, err
	}

	feedreader.Sources = feeds
	return feedreader, nil
}

func (fr *FeedReader) Fetch() error {
	feeds := make([]Feed, 0)
	count := len(fr.Sources)

	for i, url := range fr.Sources {
		items, err := fetchFeed(url)

		if err != nil {
			fmt.Printf("%d/%d: Failed %s %s\n", i, count, url, err)
			continue
		}

		feed := Feed{
			URL:   url,
			Items: items,
		}
		feeds = append(feeds, feed)
		fmt.Printf("%d/%d: %s\n", i, count, url)
	}

	fr.Feeds = feeds
	return nil
}

func (fr *FeedReader) Store(outDir string, location *time.Location) error {
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err := os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	dayLocation := time.Now().In(location)
	day := dayLocation.Format("2-1-2006")
	feedFile := outDir + day + ".json"
	feeds := fr.Feeds
	if _, err := os.Stat(feedFile); !os.IsNotExist(err) {
		feedsFile, err := ioutil.ReadFile(feedFile)
		if err != nil {
			return err
		}

		var oldFeeds []Feed
		err = json.Unmarshal(feedsFile, &oldFeeds)
		if err != nil {
			return err
		}

		feeds = merge(feeds, oldFeeds)
	}

	jsonFeeds, err := json.Marshal(feeds)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(feedFile, jsonFeeds, 0644)
	return err
}

func fetchFeed(url string) ([]*FeedItem, error) {
	feedParser := gofeed.NewParser()
	feed, err := feedParser.ParseURL(url)
	if err != nil {
		return nil, err
	}

	items := make([]*FeedItem, len(feed.Items))
	for i, item := range feed.Items {
		newItem := FeedItem{
			Title:     item.Title,
			Content:   item.Description,
			URL:       item.Link,
			Published: item.Published,
			GUID:      item.GUID,
		}

		err := newItem.validate()
		if err != nil {
			return nil, err
		}

		items[i] = &newItem
	}
	return items, nil
}

func loadFeeds(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	feeds := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		feeds = append(feeds, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return feeds, nil
}

func merge(newFeeds []Feed, oldFeeds []Feed) []Feed {
	oldFeedsMap := make(map[string][]*FeedItem, len(oldFeeds))
	feeds := make([]Feed, len(newFeeds))

	for _, feed := range oldFeeds {
		oldFeedsMap[feed.URL] = feed.Items
	}

	for _, feed := range newFeeds {
		if oldFeedsMap[feed.URL] != nil {
			items := removeDuplicates(feed.Items, oldFeedsMap[feed.URL])
			newFeed := Feed{
				URL:   feed.URL,
				Items: items,
			}
			feeds = append(feeds, newFeed)
		} else {
			feeds = append(feeds, feed)
		}
	}

	return feeds
}

func removeDuplicates(newItems []*FeedItem, oldItems []*FeedItem) []*FeedItem {
	found := make(map[string]bool)

	for _, item := range oldItems {
		found[item.GUID] = true
	}

	for _, item := range newItems {
		if !found[item.GUID] {
			oldItems = append(oldItems, item)
		}
	}

	return oldItems
}

func (item *FeedItem) validate() error {
	if item.URL == "" {
		return fmt.Errorf("Feed item contains no url: %s", item)
	}

	if item.GUID == "" {
		item.GUID = item.URL
	}

	if item.Published == "" {
		item.Published = time.Now().String()
	}

	return nil
}