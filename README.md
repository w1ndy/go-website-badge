# go-website-badge
> Generates status badges for websites

![](https://img.shields.io/badge/status-up-success.svg) ![](https://img.shields.io/badge/status-down-critical.svg)

## Usage

Create a ``config.json`` accordingly and then

```
$ docker run -it --rm -v $(pwd)/config.json:/app/config.json -p 8080:8080 w1ndy/go-website-badge
```

Visit http://localhost:8080/{Identifier} (e.g., http://localhost:8080/bing) for badges. These badges are generated with [shields.io](https://shields.io).


Licensed under [Anti 996](https://github.com/996icu/996.ICU/blob/master/LICENSE).
Copyright (c) 2019 [Project Contributors](https://github.com/w1ndy/12306.ics/graphs/contributors).