module github.com/tinkernels/doh-proxy/v5

go 1.14

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.0
	github.com/emirpasic/gods v1.12.0
	github.com/miekg/dns v1.1.35
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/sirupsen/logrus v1.7.0
	github.com/zput/zxcTool v1.3.6
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20201101102859-da207088b7d1 // indirect
)

replace github.com/sirupsen/logrus v1.7.0 => github.com/tinkernels/logrus v1.7.1-0.20201103164625-e081dd4f4900

replace github.com/zput/zxcTool v1.3.6 => github.com/tinkernels/zxcTool v1.3.7-0.20210207154812-aca5af524a3a
