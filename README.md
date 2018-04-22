# Homeblocker

Homeblocker is an easy to configure, but flexible DNS blocker for home networks.

It will boost your productivity and limit your kid's compulsive surfing, too.

## Features

* Block exact domains or wildcard subdomains.
* Flexible schedule with familiar [crontab](https://en.wikipedia.org/wiki/Cron#Overview) syntax.
* Block one set of domains on one schedule, and another set on another schedule (and so on.)
* Deployed on premises for absolute privacy (apart from the upstream DNS provider, of course.)
* Does not affect performance.

## Installation

I suggest you install it on a [Raspberry Pi](https://www.raspberrypi.org/products/) (or analog). The CPU and RAM requirements are minimal, however, you should use a wired network connection.

I use homeblocker on an [ODROID U3](http://www.hardkernel.com/main/products/prdt_info.php?g_code=g138745696275) and I use my [ASUS RT-AC56U](https://www.asus.com/Networking/RTAC56U/) router's [DNS filtering](https://github.com/RMerl/asuswrt-merlin/wiki/DNS-Filter) feature to route all DNS requests to the homeblocker instance.

Of course, you can install it on any always-on machine:

* you can probably install it on the router itself, if you know how.
* if you only want blocking for your own computer, install it right there.
* you can deploy it to the internet (though performance will probably suffer, compared to a distributed DNS provider).

The instructions are:

* Unpack
* Copy `homeblocker.example.yml` to `homeblocker.yml` and edit to your preferences.
* Boot up

## Configuration manual

See [homeblocker.example.yml](./homeblocker.example.yml) for detailed configuration instructions.

---

Made by [Leonid Shevtsov](https://leonid.shevtsov.me)
