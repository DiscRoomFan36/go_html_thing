# Royal Road HTML URL to Markdown

A command line tool to download entire fictions from [Royal Road](https://www.royalroad.com) (a web novel site), and converts them into markdown. (while also removing the "this fiction wasn't read on royal road hidden text")

## A Warning

This is a warning, to all those who may follow in my footsteps, do **NOT** attempt to parse HTML into a structure.

It will not work, and I finally understand why everybody makes a running parser out of the HTML. Not just because "we should display the page while its loading". But it's fundamentally broken, and no website will ever give you correct HTML. (We should just delete \<br>. its literally useless, and if you need a line break, just add \<p>)

Oh, also the parser is prone to Assert(), not good when every little thing in HTML is prone to edge cases

## Quick start

```
$ go build

# usage is currently terrible, deal with it.
$ ./html_thing help

# these are the only real commands
$ ./html_thing chapter [CHAPTER URL]
$ ./html_thing fiction [FICTION URL]
```
