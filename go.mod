module github.com/tikernels/doh-proxy

go 1.14

require (
	github.com/antonfisher/nested-logrus-formatter v1.0.3
	github.com/emirpasic/gods v1.12.0
	github.com/miekg/dns v1.1.29
	github.com/sirupsen/logrus v1.6.0
	github.com/zput/zxcTool v1.3.6
	golang.org/x/crypto v0.0.0-20200423211502-4bdfaf469ed5 // indirect
	golang.org/x/net v0.0.0-20200421231249-e086a090c8fd // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20200420163511-1957bb5e6d1f // indirect
)

replace github.com/sirupsen/logrus v1.6.0 => github.com/tinkernels/logrus v1.6.2
