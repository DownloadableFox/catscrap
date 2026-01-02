package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const START_PAGE = "https://warriors.fandom.com/wiki/Squirrelstar"
const THREADS = 8

var search_queue chan string
var work sync.WaitGroup

var visited map[string]bool
var visited_lock sync.Mutex

var connection_map map[string][]string
var connection_lock sync.Mutex

func SetVisited(url string) {
	visited_lock.Lock()
	visited[url] = true
	visited_lock.Unlock()
}

func GetVisited(url string) bool {
	visited_lock.Lock()
	defer visited_lock.Unlock()

	if _, ok := visited[url]; ok {
		return true
	}

	return false
}

func Connect(src, dst string) {
	connection_lock.Lock()
	defer connection_lock.Unlock()

	if _, ok := connection_map[src]; !ok {
		connection_map[src] = make([]string, 0)
	}

	connection_map[src] = append(connection_map[src], dst)
}

func SearchWorker(thread_id int) {
	fmt.Printf("Started worker thread %d...\n", thread_id)

	for query_url := range search_queue {
		// Check if the page has already been visited, if not
		// it is skipped
		if GetVisited(query_url) {
			work.Done()
			continue
		}

		SetVisited(query_url)

		// The page is created
		fmt.Printf("Obtaining data for page %q\n", query_url)
		page, err := LoadPage(query_url)
		if err != nil {
			fmt.Println(err)
			work.Done()
			continue
		}

		if !page.IsCharacter {
			fmt.Printf("Page %q is not a character! Skipping...\n", page.Name)
			work.Done()
			continue
		}

		for _, connection := range page.Connections {
			if CheckValidUrl(connection) && !GetVisited(connection) {
				// The connection is done				
				connection_name, _ := GetPageName(connection)
				Connect(page.Name, connection_name)
				
				// The character connected is added to the queue
				work.Add(1)
				search_queue <- connection
			}
		}

		work.Done()
	}

	fmt.Printf("Worker thread %d has closed...\n", thread_id)
}

func main() {
	// The search queue is created
	search_queue = make(chan string, 100000)

	// The visited set is also created
	visited = make(map[string]bool)

	// Connection tree is created
	connection_map = make(map[string][]string)

	// Initial setup
	work.Add(1)
	search_queue <- START_PAGE

	// Worker creation
	wg := sync.WaitGroup{}
	for i := range THREADS {
		wg.Go(func() {
			SearchWorker(i)
		})
	}

	// No more work available
	work.Wait()
	close(search_queue)

	// Thread exiting
	wg.Wait()
	fmt.Println("All threads have stopped. Writing connection tree")

	// Write tree
	tree, err := json.MarshalIndent(connection_map, "", "\t")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile("connections.json", tree, 0644); err != nil {
		panic(err)
	}
}
