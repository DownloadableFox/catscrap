package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const StartPage = "https://warriors.fandom.com/wiki/Acorntail"
const WorkerCount = 8

var searchQueue chan string
var workCounter sync.WaitGroup

var visited map[string]bool
var visitedMutex sync.Mutex

var connectionTree map[string][]string
var connectionMutex sync.Mutex

func SetVisited(url string) {
	visitedMutex.Lock()
	visited[url] = true
	visitedMutex.Unlock()
}

func GetVisited(url string) bool {
	visitedMutex.Lock()
	defer visitedMutex.Unlock()

	if _, ok := visited[url]; ok {
		return true
	}

	return false
}

func Connect(src, dst string) {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()

	if _, ok := connectionTree[src]; !ok {
		connectionTree[src] = make([]string, 0)
	}

	connectionTree[src] = append(connectionTree[src], dst)
}

func SearchWorker(id int) {
	fmt.Printf("Started worker thread %d...\n", id)

	for queryUrl := range searchQueue {
		// Check if the page has already been visited, if not
		// it is skipped
		if GetVisited(queryUrl) {
			workCounter.Done()
			continue
		}

		SetVisited(queryUrl)

		// The page is created
		fmt.Printf("Obtaining data for page %q\n", queryUrl)
		page, err := LoadPage(queryUrl)
		if err != nil {
			fmt.Println(err)
			workCounter.Done()
			continue
		}

		if !page.IsCharacter {
			fmt.Printf("Page %q is not a character! Skipping...\n", page.Name)
			workCounter.Done()
			continue
		}

		for _, connection := range page.Connections {
			if CheckValidUrl(connection) && !GetVisited(connection) {
				// The connection is done				
				connectionName, _ := GetPageName(connection)
				Connect(page.Name, connectionName)
				
				// The character connected is added to the queue
				workCounter.Add(1)
				searchQueue <- connection
			}
		}

		workCounter.Done()
	}

	fmt.Printf("Worker thread %d has closed...\n", id)
}

func main() {
	// The search queue is created
	searchQueue = make(chan string, 100000)

	// The visited set is also created
	visited = make(map[string]bool)

	// Connection tree is created
	connectionTree = make(map[string][]string)

	// Initial setup
	workCounter.Add(1)
	searchQueue <- StartPage

	// Worker creation
	wg := sync.WaitGroup{}
	for i := range WorkerCount {
		wg.Go(func() {
			SearchWorker(i)
		})
	}

	// No more work available
	workCounter.Wait()
	close(searchQueue)

	// Thread exiting
	wg.Wait()
	fmt.Println("All threads have stopped. Writing connection tree...")

	// Tree sanitization (removal of non characters in connections)
	for key, connections := range connectionTree {
		sanitizedConnections := make([]string, 0)
		for _, character := range connections {
			if _, exists := connectionTree[character]; exists {
				sanitizedConnections = append(sanitizedConnections, character)
			}
		}
		
		connectionTree[key] = sanitizedConnections
	}

	// Write tree
	tree, err := json.MarshalIndent(connectionTree, "", "\t")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile("connections.json", tree, 0644); err != nil {
		panic(err)
	}
}
