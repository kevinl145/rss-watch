# rss-watch

A command line tool that remembers which items it has already seen in an
RSS or Atom feed, so each run only reports what's new.

## Why

I wanted to point cron at a handful of blogs and mailing-list feeds and get
a short list of new posts, not the full feed dumped out every time. Feed
readers already do this, but they want to own my subscription list and my
read state too. This tool does one thing: given a feed URL, print what
changed since the last check, and get out of the way.

## Install

Requires Go 1.22 or newer.

    go build -o rss-watch .

## Usage

    rss-watch https://example.com/feed.xml

The first run has nothing to compare against, so pass `--quiet` to record
the current items without printing them:

    rss-watch --quiet https://example.com/feed.xml

Every run after that only prints items that weren't seen before:

    $ rss-watch https://example.com/feed.xml
    2 new item(s) in Example Blog:
    - A New Post
      https://example.com/posts/a-new-post
    - Another Post
      https://example.com/posts/another-post

Pass `--json` for machine-readable output, for piping into another script:

    $ rss-watch --json https://example.com/feed.xml
    {
      "feed": "https://example.com/feed.xml",
      "title": "Example Blog",
      "new_items": [
        {
          "id": "https://example.com/posts/a-new-post",
          "title": "A New Post",
          "link": "https://example.com/posts/a-new-post",
          "published": "Sat, 22 Aug 2026 09:00:00 GMT"
        }
      ]
    }

To see which feeds are already tracked, and how many items are recorded as
seen for each, use `--list`. It reads the state file and exits without
fetching anything, so it doesn't need a feed URL:

    $ rss-watch --list
    https://example.com/feed.xml
      12 item(s) seen, last checked 2026-09-06T08:00:00Z

`--list` also honors `--json`.

To watch several feeds in one run, put one feed URL per line in a text file
and pass it with `--config` instead of a feed argument:

    # feeds.txt
    https://example.com/feed.xml
    https://another-example.com/index.xml

    $ rss-watch --config feeds.txt
    https://example.com/feed.xml:
    2 new item(s) in Example Blog:
    - A New Post
      https://example.com/posts/a-new-post
    - Another Post
      https://example.com/posts/another-post
    https://another-example.com/index.xml:
    no new items

Blank lines and lines starting with `#` are ignored, and a URL listed twice
is only fetched once. All feeds share the same state file, `--quiet`,
`--timeout`, and `--json` flags as a single-feed run. If any feed in the
list fails to fetch or parse, an error is printed for that feed and the rest
still run; the command then exits non-zero.

### Flags

    --config string  path to a file listing feed URLs to watch, one per line
    --json           output new items as JSON instead of plain text
    --list           list feeds tracked in the state file and exit
    --quiet          record current items as seen without printing anything
    --state string   path to the state file (default ~/.rss-watch/state.json)
    --timeout        HTTP timeout when fetching the feed (default 15s)

## State

Seen items are tracked per feed URL in a single JSON file. Delete the file,
or the entry for one feed, to make everything look new again.

Each feed entry also remembers the `ETag` and `Last-Modified` values from
the last response, if the server sent them, and uses them on the next run
as `If-None-Match` / `If-Modified-Since`. A server that replies `304 Not
Modified` means the feed body isn't re-fetched or re-parsed at all.

## Supported formats

RSS 2.0, Atom, and RSS 1.0 / RDF.

## License

MIT, see LICENSE.
