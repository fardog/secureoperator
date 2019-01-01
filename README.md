# secureoperator

[![Build Status](https://travis-ci.org/fardog/secureoperator.svg?branch=master)](https://travis-ci.org/fardog/secureoperator)
[![](https://godoc.org/github.com/fardog/secureoperator?status.svg)](https://godoc.org/github.com/fardog/secureoperator)

A DNS-protocol proxy for [DNS-over-HTTPS][dnsoverhttps]: allows you to run a
server on your local network which responds to DNS queries, but requests records
across the internet using HTTPS.

It's known to work with the following providers:

* [Google][dnsoverhttps] - Well tested and configured by default
* [Cloudflare][] _(Beta)_ - May be used by passing the `--cloudflare` flag
* [Quad9][] _(Beta)_ - May be used by passing the `--quad9' flag

If you're interested in a more roll-your-own-DNS system, you might look at
[dnoxy][], a sibling project to secureoperator which allows running your own
DNS-over-HTTPS servers.

## Installation

You may retrieve binaries from [the releases page][releases], or install using
`go get`:

```
go get -u github.com/fardog/secureoperator/cmd/secure-operator
```

Then either run the binary you downloaded, or the built package:

```
secure-operator
```

This will start a DNS server listening on TCP and UDP at `:53`. For usage
information, run `secure-operator --help`.

**Note:** Running a service on port `53` requires administrative privileges on
most systems.

### Docker

There is a [Docker][docker] image available for secureoperator:

```
docker pull fardog/secureoperator
```

The `latest` tag will always be the build from the `master` branch. If you wish
to use one of the stable releases, use its version tag when pulling, e.g.:

```
docker pull fardog/secureoperator:4  # latest of major version
docker pull fardog/secureoperator:4.0  # latest of minor version
docker pull fardog/secureoperator:4.0.1  # exact version
```

## Version Compatibility

This package follows [semver][] for its tagged releases. The `master` branch is
always considered stable, but may break API compatibility. If you require API
stability, either use the tagged releases or mirror on gopkg.in:

```
go get -u gopkg.in/fardog/secureoperator.v4
```

## Caching

secureoperator _does not perform any caching_; each request to it causes a
matching request to the upstream DNS-over-HTTPS server to be made. It's
recommended that you place secureoperator behind a caching DNS server such as
[dnsmasq][] on your local network.

An simple example setup is [described on the wiki][wiki-setup]. Please feel free
to contribute additional setups if you are running secureoperator in your
environment.

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
      
Information on the usage of these options is available with
`secure-operator --help`. 
  
## Help Wanted

secureoperator could be greatly enhanced by community contributions! The
following areas could use work:

* More thorough unit tests
* Installable packages for your favorite Linux distributions
* Documentation on deploying secureoperator to a local network

### Known Issues

Cloudflare is not fully tested yet; it should work for common cases, however: 

* EDNS is not supported; this is an intentional choice by Cloudflare, which
  means any EDNS setting you provide when using Cloudflare as a provider will
  be silently ignored.

For a production environment, the Google provider (default) is your best option
today. If you're brave, please test Cloudflare and [report any issues][issues]!

## Acknowledgments

This owes heavily to the following work:

* https://github.com/miekg/dns
* https://github.com/wrouesnel/dns-over-https-proxy
* https://github.com/StalkR/dns-reverse-proxy

## License

```
   Copyright 2018 Nathan Wittstock

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0
```

[dnsoverhttps]: https://developers.google.com/speed/public-dns/docs/dns-over-https
[googlednspriv]: https://developers.google.com/speed/public-dns/privacy
[cloudflarednspriv]: https://developers.cloudflare.com/1.1.1.1/commitment-to-privacy/
[releases]: https://github.com/fardog/secureoperator/releases
[docker]: https://www.docker.com/
[issues]: https://github.com/fardog/secureoperator/issues
[semver]: http://semver.org/
[wiki-setup]: https://github.com/fardog/secureoperator/wiki/Setting-up-dnsmasq-with-secureoperator
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[cloudflare]: https://1.1.1.1/
[quad9]: https://www.quad9.net/
[dnoxy]: https://github.com/fardog/dnoxy
