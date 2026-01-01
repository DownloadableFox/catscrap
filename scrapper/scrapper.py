import time
from collections import deque
from page import Page

START_PAGE = "https://warriors.fandom.com/wiki/Squirrelstar"

# Search queue
to_search = deque()
to_search.append(START_PAGE)

# Visited set
visited = set()

while len(to_search) > 0:
    current_url = to_search.popleft()

    # If for some reason we have already visited the page, it is skipped
    if current_url in visited:
        continue
    
    visited.add(current_url)

    # Attempts to get the page
    page = Page(current_url)
    if page.configure():
        if not page.is_character():
            print(f"Page {current_url} is not a character! Skipping...")
            continue

        # The connections are added to the queue
        print(f"Obtaining connections for {page.get_name()}")

        for neighboor in page.get_connections():
            if neighboor not in visited:
                to_search.append(neighboor)
        
        time.sleep(0.05) # Some delay added to not spam the server

    else:
        print(f"Configure failed for page: {current_url}")