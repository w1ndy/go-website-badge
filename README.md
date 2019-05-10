# go-website-badge
> Generates status badges for websites

![](https://img.shields.io/badge/status-up-success.svg) ![](https://img.shields.io/badge/last%20seen-n/a-blue.svg) ![](https://img.shields.io/badge/sla-100%25-green.svg)

## Usage

Create a ``config.json`` accordingly and then

```
$ docker run -it --rm -v $(pwd)/config.json:/app/config.json -p 8080:8080 skies457/go-website-badge
```

Visit http://localhost:8080/{Identifier} (e.g., http://localhost:8080/bing) for badges:
* Status Badges: http://localhost:8080/{Identifier}
* Last Seen Badges: http://localhost:8080/{Identifier}-lastseen
* Availability Badges: http://localhost:8080/{Identifier}-sla

These badges are generated with [shields.io](https://shields.io).


Licensed under [Anti 996](https://github.com/996icu/996.ICU/blob/master/LICENSE).
Copyright (c) 2019 [Project Contributors](https://github.com/w1ndy/go-website-badge/graphs/contributors).
