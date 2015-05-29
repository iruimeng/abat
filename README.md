# abat
Go implemented CLI cURL-like tool for humans. Abat can be used for testing, debugging, and generally interacting with HTTP servers.

Fork from [bat](https://github.com/astaxie/bat), use  a versatile HTTP load testing tool library [stress](https://github.com/buaazp/stress). Thanks to the author, astaxie buzzap.


- [Main Features](#main-features)
- [Installation](#installation)
- [Usage](#usage)
- [HTTP Method](#http-method)
- [Request URL](#request-url)
- [Request Items](#request-items)
- [JSON](#json)
- [Forms](#forms)
- [HTTP Headers](#http-headers)
- [Authentication](#authentication)
- [Proxies](#proxies)
- [License](#license)

## Main Features

- Expressive and intuitive syntax
- Built-in JSON support
- Forms and file uploads
- HTTPS, proxies, and authentication
- Arbitrary request data
- Custom headers

## Installation

	go get -u github.com/iruimeng/abat
	
make sure the `$GOPATH/bin` is added into `$PATH`

## Usage

Hello World:

	$ abat t.tt 

Synopsis:

	abat [METHOD] [flags] URL [ITEM [ITEM]]
	
See also `abat --help`. `abat attack -h`. `abat report -h`

### Examples

Basic settings - [HTTP method](#http-method), [HTTP headers](#http-headers) and [JSON](#json) data:

	$ abat PUT example.org X-API-Token:123 name=John

Any custom HTTP method (such as WebDAV, etc.):

	$ abat -method=PROPFIND example.org name=John

Submitting forms:

	$ abat -form=true POST example.org hello=World
	
See the request that is being sent using one of the output options:

	$ abat -v example.org

Use Github API to post a comment on an issue with authentication:

	$ abat -a USERNAME POST https://api.github.com/repos/iruimeng/abat/issues/1/comments body='abat is awesome!'

Upload a file using redirected input:

	$ abat example.org < file.json
	
Download a file and save it via redirected output:

	$ abat example.org/file > file
	
Download a file wget style:

	$ abat -download=true example.org/file

Set a custom Host header to work around missing DNS records:

	$ abat localhost:8000 Host:example.com
	
Following is the detailed documentation. It covers the command syntax, advanced usage, and also features additional examples.

### attack

````
$ abat attack -h
Usage of abat attack:
  -body="": Requests body file
  -c=10: Concurrency level
  -duration=10s: Duration of the test
  -header=: Request header
  -laddr=0.0.0.0: Local IP address
  -n=1000: Requests number
  -ordering="random": Attack ordering [sequential, random]
  -output="result.json": Output file
  -rate=50: Requests per second
  -redirects=10: Number of redirects to follow
  -targets="stdin": Targets file
  -timeout=0: Requests timeout
````
#### -targets
Specifies the attack targets in a line separated file, defaulting to stdin. The format should be as follows.
````
	GET www.baidu.com
	HEAD t.tt
````

### report
````
$ abat report -h
Usage of abat report:
  -input="stdin": Input files (comma separated)
  -output="stdout": Output file
  -reporter="text": Reporter [text, json, plot]

$ abat attack -c=10 -targets=target.txt  
$ cat result.json | abat report -reporter=plot > o.html
````

#### -input
Specifies the input files to generate the report of, defaulting to stdin.
These are the output of stress attack. You can specify more than one (comma
separated) and they will be merged and sorted before being used by the
reports.

#### -output
Specifies the output file to which the report will be written to.

#### -reporter
Specifies the kind of report to be generated. It defaults to text.

##### text
````
Requests      [total]                   1200
Duration      [total]                   1.998307684s
Latencies     [mean, 50, 95, 99, max]   223.340085ms, 240.12234ms, 326.913687ms, 416.537743ms, 7.788103259s
Bytes In      [total, mean]             3714690, 3095.57
Bytes Out     [total, mean]             0, 0.00
Success       [ratio]                   55.42%
Status Codes  [code:count]              0:535  200:665
Error Set:
Get http://localhost:6060: dial tcp 127.0.0.1:6060: connection refused
Get http://localhost:6060: read tcp 127.0.0.1:6060: connection reset by peer
Get http://localhost:6060: dial tcp 127.0.0.1:6060: connection reset by peer
Get http://localhost:6060: write tcp 127.0.0.1:6060: broken pipe
Get http://localhost:6060: net/http: transport closed before response was received
Get http://localhost:6060: http: can't write HTTP request on broken connection
````

## HTTP Method
The name of the HTTP method comes right before the URL argument:

	$ abat DELETE example.org/todos/7
	
which looks similar to the actual Request-Line that is sent:

DELETE /todos/7 HTTP/1.1

When the METHOD argument is omitted from the command, abat defaults to either GET (if there is no request data) or POST (with request data).

## Request URL
The only information abat needs to perform a request is a URL. The default scheme is, somewhat unsurprisingly, http://, and can be omitted from the argument – `abat example.org` works just fine.

Additionally, curl-like shorthand for localhost is supported. This means that, for example :3000 would expand to http://localhost:3000 If the port is omitted, then port 80 is assumed.

	$ abat :/foo

	GET /foo HTTP/1.1
	Host: localhost

	$ abat :3000/bar
	
	GET /bar HTTP/1.1
	Host: localhost:3000

	$ abat :

	GET / HTTP/1.1
	Host: localhost

If you find yourself manually constructing URLs with query string parameters on the terminal, you may appreciate the `param=value` syntax for appending URL parameters so that you don't have to worry about escaping the & separators. To search for abat on Google Images you could use this command:

	$ abat GET www.google.com search=abat tbm=isch

	GET /?search=abat&tbm=isch HTTP/1.1

## Request Items
There are a few different request item types that provide a convenient mechanism for specifying HTTP headers, simple JSON and form data, files, and URL parameters.

They are key/value pairs specified after the URL. All have in common that they become part of the actual request that is sent and that their type is distinguished only by the separator used: `:`, `=`, `:=`, `@`, `=@`, and `:=@`. The ones with an `@` expect a file path as value.


|       Item Type         |	          Description           |
| ------------------------| ------------------------------ | 
|HTTP Headers `Name:Value`|Arbitrary HTTP header, e.g. `X-API-Token:123`.|
|Data Fields `field=value`|Request data fields to be serialized as a JSON object (default), or to be form-encoded (--form, -f).|
|Form File Fields `field@/dir/file`|Only available with `-form`, `-f`. For example `screenshot@~/Pictures/img.png`. The presence of a file field results in a `multipart/form-data` request.|
|Form Fields from file `field=@file.txt`|read content from file as value|
|Raw JSON fields `field:=json`, `field:=@file.json`|Useful when sending JSON and one or more fields need to be a Boolean, Number, nested Object, or an Array, e.g., meals:='["ham","spam"]' or pies:=[1,2,3] (note the quotes).|

You can use `\` to escape characters that shouldn't be used as separators (or parts thereof). For instance, foo\==bar will become a data key/value pair (foo= and bar) instead of a URL parameter.

You can also quote values, e.g. `foo="bar baz"`.
## JSON
JSON is the lingua franca of modern web services and it is also the implicit content type abat by default uses:

If your command includes some data items, they are serialized as a JSON object by default. abat also automatically sets the following headers, both of which can be overwritten:

|Content-Type	| application/json |
| -------------| ----------------- |
|Accept	        |application/json|

You can use --json=true, -j=true to explicitly set Accept to `application/json` regardless of whether you are sending data (it's a shortcut for setting the header via the usual header notation – `abat url Accept:application/json`).

Simple example:

	$ abat PUT example.org name=John email=john@example.org
	PUT / HTTP/1.1
	Accept: application/json
	Accept-Encoding: gzip, deflate
	Content-Type: application/json
	Host: example.org
	
	{
	    "name": "John",
	    "email": "john@example.org"
	}

Non-string fields use the := separator, which allows you to embed raw JSON into the resulting object. Text and raw JSON files can also be embedded into fields using =@ and :=@:

	$ abat PUT api.example.com/person/1 \
    name=John \
    age:=29 married:=false hobbies:='["http", "pies"]' \  # Raw JSON
    description=@about-john.txt \   # Embed text file
    bookmarks:=@bookmarks.json      # Embed JSON file

	PUT /person/1 HTTP/1.1
	Accept: application/json
	Content-Type: application/json
	Host: api.example.com
	
	{
	    "age": 29,
	    "hobbies": [
	        "http",
	        "pies"
	    ],
	    "description": "John is a nice guy who likes pies.",
	    "married": false,
	    "name": "John",
	    "bookmarks": {
	        "HTTPie": "http://httpie.org",
	    }
	}
	
Send JSON data stored in a file (see redirected input for more examples):

	$ abat POST api.example.com/person/1 < person.json
	
## Forms
Submitting forms are very similar to sending JSON requests. Often the only difference is in adding the `-form=true`, `-f` option, which ensures that data fields are serialized correctly and Content-Type is set to, `application/x-www-form-urlencoded; charset=utf-8`.

It is possible to make form data the implicit content type instead of JSON via the config file.

### Regular Forms

	$ abat -f=true POST api.example.org/person/1 name='John Smith' \
    email=john@example.org

	POST /person/1 HTTP/1.1
	Content-Type: application/x-www-form-urlencoded; charset=utf-8

	name=John+Smith&email=john%40example.org

### File Upload Forms

If one or more file fields is present, the serialization and content type is `multipart/form-data`:

	$ abat -f=true POST example.com/jobs name='John Smith' cv@~/Documents/cv.pdf
	
The request above is the same as if the following HTML form were submitted:

```
<form enctype="multipart/form-data" method="post" action="http://example.com/jobs">
    <input type="text" name="name" />
    <input type="file" name="cv" />
</form>
```

Note that `@` is used to simulate a file upload form field.

## HTTP Headers
To set custom headers you can use the Header:Value notation:

	$ abat example.org  User-Agent:Bacon/1.0  'Cookie:valued-visitor=yes;foo=bar'  \
    X-Foo:Bar  Referer:http://beego.me/

	GET / HTTP/1.1
	Accept: */*
	Accept-Encoding: gzip, deflate
	Cookie: valued-visitor=yes;foo=bar
	Host: example.org
	Referer: http://beego.me/
	User-Agent: Bacon/1.0
	X-Foo: Bar
	
There are a couple of default headers that abat sets:

	GET / HTTP/1.1
	Accept: */*
	Accept-Encoding: gzip, deflate
	User-Agent: abat/<version>
	Host: <taken-from-URL>

Any of the default headers can be overwritten.

# Authentication
Basic auth:

	$ abat -a=username:password example.org

# Proxies
You can specify proxies to be used through the --proxy argument for each protocol (which is included in the value in case of redirects across protocols):

	$ abat --proxy=http://10.10.1.10:3128 example.org
	
With Basic authentication:

	$ abat --proxy=http://user:pass@10.10.1.10:3128 example.org
	
You can also configure proxies by environment variables HTTP_PROXY and HTTPS_PROXY, and the underlying Requests library will pick them up as well. If you want to disable proxies configured through the environment variables for certain hosts, you can specify them in NO_PROXY.

In your ~/.bash_profile:

	export HTTP_PROXY=http://127.0.0.1:1010
	export HTTPS_PROXY=https://127.0.0.1:1011
	export NO_PROXY=localhost,example.com





#License

Apache License
http://www.apache.org/licenses/LICENSE-2.0
