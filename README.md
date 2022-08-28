# Booker

Copyright 2022 Hal Canary

Use of this program is governed by the file [LICENSE](./LICENSE).

## Usage

    ./booker [FLAGS] URL [MORE_URLS]

    -over
          force overwrite of output file
    -send
          also send via email

## Description

Booker takes a list of URLs as arguments.  It then attempts to download the
books at each URL and save it as an EPUB file inn the `~/ebooks` directory.

The name of the file will be `{TITLE}_{TIME}.epub`, where `{TIME}` is the
modification time of the most recently changed chapter of the book.

If that file already exists, nothing is done, unless the `-over` flag is set.

Chapters are cached, so some chapter updates are ignored.  To clear the cache,
`rm -r $CACHEHOME/urlcache`, where `$CACHEHOME` is `$XDG_CACHE_HOME` or
`~/.cache` or `~/Library/Caches` or `%LocalAppData%`, depending on your OS.

## Email

If the `-send` flag is set, any new EPUB files are emailed to the address
listed in the file `~/.ebook_address`, if it exists.  SMTP credentials are read
from the file `~/.email_secrets.json`.  This file has the following format:

    {
      "SMTP_HOST": "host.example.com",
      "SMTP_USER": "example@example.com",
      "SMTP_PASS": "password123",
      "FROM_ADDR": "Example <example@example.com>",
    }

## Internal APIs

See [./docs](./docs).
