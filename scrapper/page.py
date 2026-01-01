import requests
import re
from typing import Optional
from bs4 import BeautifulSoup
from urllib.parse import unquote

CACHE_DIRECTORY = '.cache'
CHARACTER_URL_PATTERN = r"^https:\/\/warriors\.fandom\.com\/wiki\/([^\/:]+)$"

def load_cached(name: str) -> Optional[str]:
    path = f"{CACHE_DIRECTORY}/{name}.html"

    try:
        with open(path, 'r') as file:
            return file.read()

    except FileNotFoundError:
        return None

    except Exception as e:
        printf(f"An exception occured whilst reading file: {path}")
        return None

def store_cached(name: str, content: str):
    path = f"{CACHE_DIRECTORY}/{name}.html"

    try:
        with open(path, 'w') as file:
            file.write(content)

    except Exception as e:
        print(f"An exeption occured whilst storing file: {path}")

def get_page_name(url: str) -> Optional[str]:
    pattern_match = re.match(CHARACTER_URL_PATTERN, url)

    if pattern_match:
        return unquote(pattern_match.group(1)) # Removes HTML encoding

    return None

class Page:
    def __init__(self, url: str):
        self.url = url

    def configure(self) -> bool:
        self.page_name = get_page_name(self.url)

        # The page name is validated, if it doesn't match the string it's
        # straight up just not a character.
        if not self.page_name:
            print(f"Invalid character page url for: {self.url}")
            return False
        
        # The cached version is loaded when possible
        cached_page = load_cached(self.page_name)
        if cached_page:
            self.contents = cached_page
            return True
        
        # If all fails, the page is downloaded directly from Fandom
        response = requests.get(self.url)
        if response.status_code == 200:
            self.contents = response.text
            store_cached(self.page_name, self.contents)
            return True

        else:
            print(f"Request for {self.url} failed with status code: {response.status_code}")
            return False

    def is_character(self) -> bool:
        soup = BeautifulSoup(self.contents, "html.parser")

        for a in soup.select("div.page-header__categories a"):
            if a.get("href") == "/wiki/Category:Characters":
                return True

        return False

    def get_name(self) -> str:
        return self.page_name

    def get_connections(self) -> list[str]:
        if not self.contents:
            return []

        soup = BeautifulSoup(self.contents, "html.parser")

        urls = set()

        for a in soup.find_all("a", href=True):
            url = a["href"]

            # If the page matches the typical
            # page style it can be considered a character
            if get_page_name(url):
                urls.add(url)

        return list(urls)
