# secureoperator

[![Build Status](https://travis-ci.org/fardog/secureoperator.svg?branch=master)](https://travis-ci.org/fardog/secureoperator)
[![](https://godoc.org/github.com/fardog/secureoperator?status.svg)](https://godoc.org/github.com/fardog/secureoperator)

A DNS-protocol proxy for Google's [DNS-over-HTTPS][dnsoverhttps]: allows you to
run a server on your local network which responds to DNS queries, but requests
records across the internet using HTTPS.

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
docker pull fardog/secureoperator:v1.0.3
```

## Security

Note that while DNS requests are made over HTTPS, this does not imply "secure";
consider the following:

* You must trust Google with your requests, see
  [their privacy statement][googlednspriv] for further details.
* The initial lookup for the Google DNS endpoint happens over plain DNS using
  your locally configured DNS resolver; there are plans to mitigate this in the
  future, but at least _one_ DNS request will be sent unsecured.
  
## Caveats/TODO

* Currently only the following records are supported: `A, AAAA, CNAME, MX`
* Padding is not very smart; it just always pads to 1024 characters, and fails
  if the URL would've been larger than that
* More thorough tests should be written
* No caching is implemented, and probably never will. If you need caching, put
  your `secure-operator` server behind another DNS server which provides
  caching.

## Acknowledgments

This owes heavily to the following work:

* https://github.com/miekg/dns
* https://github.com/wrouesnel/dns-over-https-proxy
* https://github.com/StalkR/dns-reverse-proxy

## License

```
   Copyright 2017 Nathan Wittstock

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0
```

[dnsoverhttps]: https://developers.google.com/speed/public-dns/docs/dns-over-https
[googlednspriv]: https://developers.google.com/speed/public-dns/privacy
[releases]: https://github.com/fardog/secureoperator/releases
[docker]: https://www.docker.com/
[issues]: https://github.com/fardog/secureoperator/issues
