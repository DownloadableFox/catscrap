package main

import (
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const CacheDirectory = ".cache"

var RegexCharacterURL = regexp.MustCompile(`https:\/\/warriors\.fandom\.com\/wiki\/([^\/:\?#]+)$`)
var RegexCharacterHREF = regexp.MustCompile(`^\/wiki\/([^\/:]+)$`)

var ErrInvalidCharacterUrl = errors.New("invalid character URL")
var ErrPageIsNotCharacter = errors.New("page given is not a character")

func loadCached(name string) (string, error) {
	path := fmt.Sprintf("%s/%s.html", CacheDirectory, name)
	contents, err := os.ReadFile(path)
	return string(contents), err
}

func storeCached(name, content string) error {
	path := fmt.Sprintf("%s/%s.html", CacheDirectory, name)
	return os.WriteFile(path, []byte(content), 0644)
}

func downloadPage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get page %q: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read page's body %q: %w", url, err)
	}

	return string(body), nil
}

func GetPageName(url string) (string, error) {
	match := RegexCharacterURL.FindStringSubmatch(url)
	if len(match) < 1 {
		return "", ErrInvalidCharacterUrl
	}

	return html.UnescapeString(match[1]), nil
}

func CheckValidUrl(url string) bool {
	_, err := GetPageName(url)
	return err == nil
}

type Page struct {
	URL         string
	Name        string
	IsCharacter bool
	Connections []string
	Contents    string
}

func LoadPage(url string) (*Page, error) {
	pageName, err := GetPageName(url)
	if err != nil {
		return nil, err
	}

	cachedContents, err := loadCached(pageName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fmt.Printf("Failed to load cache for character %s: %v\n", pageName, err)
			return nil, err
		}

		contents, err := downloadPage(url)
		if err != nil {
			return nil, err
		}

		if err := storeCached(pageName, contents); err != nil {
			fmt.Printf("failed to page %q to cache: %v\n", pageName, err)
		}

		return LoadPageFromContents(url, contents)
	}

	return LoadPageFromContents(url, cachedContents)
}

func LoadPageFromContents(url, contents string) (*Page, error) {
	pageName, err := GetPageName(url)
	if err != nil {
		return nil, err
	}

	isCharacter := false
	connections := make([]string, 0)
	connectionSet := make(map[string]bool)

	reader := strings.NewReader(contents)
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	document.Find("div.page-header__categories a").Each(func(_ int, a *goquery.Selection) {
		if value, _ := a.Attr("href"); value == "/wiki/Category:Characters" {
			isCharacter = true
		}
	})

	document.Find("main a").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		matches := RegexCharacterHREF.MatchString(href)
		_, visited := connectionSet[href]

		if matches && !visited {
			connections = append(connections, fmt.Sprintf("https://warriors.fandom.com%s", href))
			connectionSet[href] = true
		}
	})

	return &Page{
		URL:         url,
		Name:        pageName,
		IsCharacter: isCharacter,
		Connections: connections,
		Contents:    contents,
	}, nil
}
