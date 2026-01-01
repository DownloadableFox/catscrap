import time
import threading
from collections import deque 
from page import Page

START_PAGE = "https://warriors.fandom.com/wiki/Squirrelstar"
THREADS = 8
EMPTY_WAIT = 1.0  # seconds to wait before checking again when queue is empty

# Search queue
to_search = deque()
to_search.append(START_PAGE)

# Visited set
visited = set()

# Threading locks
search_mutex = threading.Lock()
visited_mutex = threading.Lock()


def search_thread(thread_id: int):
    print(f"Started search thread {thread_id}...")

    while True:
        current_url = None

        with search_mutex:
            if to_search:
                current_url = to_search.popleft()

        if current_url is None:
            # Queue is empty; wait a bit before retrying
            time.sleep(EMPTY_WAIT)
            with search_mutex:
                if not to_search:
                    # Still empty after waiting, exit thread
                    break
            continue

        with visited_mutex:
            if current_url in visited:
                continue
            visited.add(current_url)

        # Attempts to get the page
        page = Page(current_url)
        if page.configure():
            if not page.is_character():
                print(f"Page {current_url} is not a character! Skipping...")
                continue

            connections = page.get_connections()
            with search_mutex, visited_mutex:
                for neighbor in connections:
                    if neighbor not in visited:
                        to_search.append(neighbor)
        else:
            print(f"Configure failed for page: {current_url}")

    print(f"Search thread {thread_id} closed!")


threads = []
for i in range(THREADS):
    t = threading.Thread(target=search_thread, args=(i,))
    t.start()
    threads.append(t)

for t in threads:
    t.join()
