module github.com/tinkernels/doh-proxy/v5

go 1.14

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/emirpasic/gods v1.12.0
	github.com/miekg/dns v1.1.43
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/sirupsen/logrus v1.7.0
	github.com/zput/zxcTool v1.3.6
)

replace github.com/sirupsen/logrus v1.7.0 => github.com/tinkernels/logrus v1.7.1-0.20201103164625-e081dd4f4900

replace github.com/zput/zxcTool v1.3.6 => github.com/tinkernels/zxcTool v1.3.7-0.20210207154812-aca5af524a3a
