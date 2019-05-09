# go-website-badge
> Generates status badges for websites

![](https://img.shields.io/badge/status-up-success.svg) ![](https://img.shields.io/badge/status-down-critical.svg) ![](https://img.shields.io/badge/last%20seen-n/a-blue.svg)

## Usage

Create a ``config.json`` accordingly and then

```
$ docker run -it --rm -v $(pwd)/config.json:/app/config.json -p 8080:8080 skies457/go-website-badge
```

Visit http://localhost:8080/{Identifier} and http://localhost:8080/{Identifier}-lastseen (e.g., http://localhost:8080/bing and http://localhost:8080/bing-lastseen) for badges. These badges are generated with [shields.io](https://shields.io).


Licensed under [Anti 996](https://github.com/996icu/996.ICU/blob/master/LICENSE).
Copyright (c) 2019 [Project Contributors](https://github.com/w1ndy/12306.ics/graphs/contributors).
