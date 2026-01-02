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

const CACHE_DIRECTORY = ".cache"

var CHARACTER_URL_PATTER = regexp.MustCompile(`https:\/\/warriors\.fandom\.com\/wiki\/([^\/:\?#]+)$`)
var CHARACTER_HREF_PATTERN = regexp.MustCompile(`^\/wiki\/([^\/:]+)$`)

var ErrInvalidCharacterUrl = errors.New("invalid character URL")
var ErrPageIsNotCharacter = errors.New("page given is not a character")

func loadCached(name string) (string, error) {
	path := fmt.Sprintf("%s/%s.html", CACHE_DIRECTORY, name)
	contents, err := os.ReadFile(path)
	return string(contents), err
}

func storeCached(name, content string) error {
	path := fmt.Sprintf("%s/%s.html", CACHE_DIRECTORY, name)
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
	match := CHARACTER_URL_PATTER.FindStringSubmatch(url)
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
	page_name, err := GetPageName(url)
	if err != nil {
		return nil, err
	}

	cached_contents, err := loadCached(page_name)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fmt.Printf("Failed to load cache for character %s: %v\n", page_name, err)
			return nil, err
		}

		contents, err := downloadPage(url)
		if err != nil {
			return nil, err
		}

		if err := storeCached(page_name, contents); err != nil {
			fmt.Printf("failed to page %q to cache: %v\n", page_name, err)
		}

		return LoadPageFromContents(url, contents)
	}

	return LoadPageFromContents(url, cached_contents)
}

func LoadPageFromContents(url, contents string) (*Page, error) {
	page_name, err := GetPageName(url)
	if err != nil {
		return nil, err
	}

	is_character := false
	connections := make([]string, 0)
	connections_set := make(map[string]bool)

	reader := strings.NewReader(contents)
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	document.Find("div.page-header__categories a").Each(func(_ int, a *goquery.Selection) {
		if value, _ := a.Attr("href"); value == "/wiki/Category:Characters" {
			is_character = true
		}
	})

	document.Find("a").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		matches := CHARACTER_HREF_PATTERN.MatchString(href)
		_, visited := connections_set[href]

		if matches && !visited {
			connections = append(connections, fmt.Sprintf("https://warriors.fandom.com%s", href))
			connections_set[href] = true
		}
	})

	return &Page{
		URL:         url,
		Name:        page_name,
		IsCharacter: is_character,
		Connections: connections,
		Contents:    contents,
	}, nil
}
