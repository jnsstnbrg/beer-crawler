# beer-crawler

> Polls [bolaget.io](https://bolaget.io/) API for beer news

This project consists of a function that polls
bolaget.io API for beer news and posts them to a Slack channel.

## Install

Install Go and follow the post-install instructions:

```sh
brew install go
```

Clone this repository:

```sh
git clone git@github.com:jnsstnbrg/beer-crawler.git $GOPATH/src/beer-crawler
```

Install the `dep` Go package management tool:

```sh
brew install dep
```

Install dependencies:

```sh
dep ensure
```

## Usage

To build the crawler run the following command:

```sh
GOOS=linux go build -o main
```

Run `main` with whatever tool you wish. Deploy to Lambda/Cloud Function, use crontab on your Raspberry, or something else entirely. All up to you. :)

## Thanks

Many thanks to larsha (https://github.com/larsha/) for the RESTful JSON API for Systembolaget and for the golang SDK.

## License

The MIT License (MIT)
