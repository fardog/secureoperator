# doh-proxy

[![Build Status](https://travis-ci.com/tinkernels/doh-proxy.svg?branch=master)](https://travis-ci.com/tinkernels/doh-proxy)
[![Go Doc](https://godoc.org/github.com/fardog/secureoperator?status.svg)](https://godoc.org/github.com/fardog/secureoperator)

A DNS-protocol proxy for [DNS-over-HTTPS][dnsoverhttps]: allows you to run a
server on your local network which responds to DNS queries, but requests records
across the internet using HTTPS.

It's known to work with the following providers:

* [Google][google doh] - Well tested with `-google` option and endpoint `https://dns.google/resolve`
* [Cloudflare][cloudflare doh]  - Tested without `-google` option
* [Quad9][quad9 doh]  - Test Wanted.

If you're interested in a more roll-your-own-DNS system, you might look at
[dnoxy][], a sibling project to secureoperator which allows running your own
DNS-over-HTTPS servers.

## Installation

You may retrieve binaries from [the releases page][releases], or install using
`go get`:

```
go get -u github.com/tinkernels/doh-proxy
```

**systemd unit file sample**

```ini
[Unit]
Description=proxy for dns over https
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/doh-proxy -google -http2 -endpoint "https://dns.google/resolve"  -endpoint-ips "8.8.8.8,8.8.4.4" -edns-subnet auto -listen 127.0.0.1:53 -no-ipv6 -cache=true -loglevel info

[Install]
WantedBy=multi-user.target
```

## Usage

Then either run the binary you downloaded, or the built package with:
```
make release
```
Information on the usage of these options is available with
```shell
__DOH_PROXY_PROGRAM_PATH__ --help

A DNS-protocol proxy for DNS-over-HTTPS service.

Usage:

  doh-proxy_macos-amd64 [options]

Options:

  -cacert string
    	CA certificate for TLS establishment
  -cache
    	Cache the dns answers (default true)
  -dns-resolver string
    	dns resolver for retrieve ip of DoH enpoint host, e.g. "8.8.8.8:53";
  -edns-subnet string
    	Specify a subnet to be sent in the edns0-client-subnet option;
    	take your own risk of privacy to use this option;
    	no: will not use edns_subnet;
    	auto: will use your current external IP address;
    	net/mask: will use specified subnet, e.g. 66.66.66.66/24.
    	        (default "auto")
  -endpoint string
    	DNS-over-HTTPS endpoint url (default "https://dns.google/dns-query")
  -endpoint-ips string
    	IPs of the DNS-over-HTTPS endpoint; if provided, endpoint lookup is
    	skipped, the TLS establishment will direct hit the "endpoint-ips". Comma
    	separated with no spaces; e.g. "8.8.8.8,8.8.4.4". One server is
    	randomly chosen for each request, failed requests are not retried.
  -google
    	Alternative google url scheme for dns.google/resolve.
  -headers value
    	Additional headers to be sent with http requests, as Key=Value; specify
    	multiple as:
    	    -header Key-1=Value-1-1 -header Key-1=Value1-2 -header Key-2=Value-2
  -http2
    	Using http2 for query connection
  -listen [host]:port
    	listen address, as [host]:port (default ":53")
  -loglevel string
    	Log level, one of: debug, info, warn, error, fatal, panic (default "info")
  -no-ipv6
    	Reply all AAAA questions with a fake answer
  -param value
    	Additional query parameters to be sent with http requests, as key=value;
    	specify multiple as:
    	    -param key1=value1-1 -param key1=value1-2 -param key2=value2
  -tcp
    	Listen on TCP (default true)
  -udp
    	Listen on UDP (default true)
```

**Note:** Running a service on port `53` requires administrative privileges on
most systems.

## Version Compatibility

This package follows [semver][] for its tagged releases. The `master` branch is
always considered stable, but may break API compatibility. If you require API
stability, either use the tagged releases or mirror on gopkg.in:

```
go get -u gopkg.in/fardog/secureoperator.v4
```

## Security

Note that while DNS requests are made over HTTPS, this does not imply "secure";
consider the following:

* You must trust the upstream provider with your requests; for your chosen
  provider, see:
  * [Google's Privacy Policy][googlednspriv]
  * [Cloudflare's Privacy Policy][cloudflarednspriv]
* The lookup for the HTTP endpoint must happen in _some_ regard, although how
  this is handled is up to you:
    * The system DNS resolver is used to look up the endpoint (default)
    * You provide a list of DNS servers to use for the endpoint lookup
    * You provide the IP address(es) to the endpoint; and no unencrypted DNS
      lookup will be performed. However if the addresses change while the
      service is running, you will need to restart the service to provide new
      addresses.

## Help Wanted

doh-proxy could be greatly enhanced by community contributions! The
following areas could use work:

* More thorough unit tests
* Installable packages for your favorite Linux distributions
* Documentation on deploying doh-proxy to a local network

### Known Issues

* **Only HTTP GET Request implemented**

* EDNS is not supported except google; this is an intentional choice by Cloudflare, which
  means any EDNS setting you provide when using Cloudflare as a provider will
  be silently ignored.

For a production environment, the Google provider (default) is your best option
today. Welcome [report any issues][issues] if you run to a panic!

## Acknowledgments

This owes heavily to the following work:

* https://github.com/miekg/dns
* https://github.com/wrouesnel/dns-over-https-proxy
* https://github.com/StalkR/dns-reverse-proxy

## License

[Apache License 2.0][license]


[googlednspriv]: https://developers.google.com/speed/public-dns/privacy
[cloudflarednspriv]: https://developers.cloudflare.com/1.1.1.1/privacy/
[releases]: https://github.com/tinkernels/doh-proxy/releases
[docker]: https://www.docker.com/
[issues]: https://github.com/tinkernels/doh-proxy/issues
[semver]: http://semver.org/
[google doh]: https://developers.google.com/speed/public-dns/docs/dns-over-https
[cloudflare doh]: https://developers.cloudflare.com/1.1.1.1/dns-over-https/
[quad9 doh]: https://www.quad9.net/
[dnoxy]: https://github.com/fardog/dnoxy
[license]: https://github.com/fardog/secureoperator/blob/master/LICENSE
[dnsoverhttps]: https://tools.ietf.org/html/rfc8484
