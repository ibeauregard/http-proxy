# HTTP Proxy


## Overview

This repo features a forward HTTP proxy used for caching purposes.

Any request made through the proxy is served from the cache if a matching cache entry is found. Otherwise, the request is sent to the upstream server. After the response is received, it is examined via its headers for cacheability, and cached if appropriate.

## Requirements

Make sure you have [Docker Compose installed](https://docs.docker.com/compose/install/).

## How to run

Make sure your Docker daemon is running.

After you cloned the repo, you can simply execute `make build run` from the project's root directory. When you build the project for the first time, expect a relatively long build, since some dependencies will have to be downloaded. Subsequent builds will be much faster.

See the Makefile for a list of `make` targets that you can use if needed.

Once running, the Forward Proxy listens and serves on [http://localhost:8080](http://localhost:8080).

## How to use

You can request any web resource through the HTTP Proxy by sending the following GET request:

`GET /?request={requested_url}`

IMPORTANT: Any ampersand character (`&`) appearing in `requested_url` needs to be URL-encoded (i.e. replaced with `%26`).

Examples:

`curl -is http://localhost:8080?request=https://go.dev`

`curl -is http://localhost:8080?request=https://github.com`

`curl -is http://localhost:8080?request=https://pokeapi.co/api/v2/pokemon?offset=20%26limit=20`

Suggestion: Pipe `curl`'s output to `less` if you want to easily view the headers.

`curl -is http://localhost:8080?request=https://go.dev | less`



## How to test
TODO [To be added shortly.]

## Key points about design and functionality

### How does the proxy determine what is cached and what is not?

The HTTP Proxy caches a response if and only if all following conditions are fulfilled:

- The response includes an `Expires` header with a date and time in the future OR a `Cache-Control` header with the `max-age` directive set to a non-zero value;
- If present, the `Cache-Control` header does not include a `Private`, `No-Cache`, or `No-Store` directive;
- The response does not include a `Set-Cookie` header.


### How is the cache actually implemented?

The HTTP Proxy's cache lives inside a dedicated directory on the server. Each cache entry corresponds to a file in that directory.

When an upstream response is deemed cacheable (see [section How does the proxy determine what is cached and what is not?](#how-does-the-proxy-determine-what-is-cached-and-what-is-not) above), a new file is created in the cache directory. The file name is simply the [MD5 checksum](https://en.wikipedia.org/wiki/MD5) of the URL requested by the client. The status line, headers and body are then written to that file.

When an entry is committed to cache, its lifespan is already known, because it can be determined either from the Cache-Control header or from the Expires headers. So we can already schedule the deletion of the cache entry with the help of Go's [time.AfterFunc](https://pkg.go.dev/time#AfterFunc), which waits for the specified time to elapse and then calls the specified function in its own goroutine. Note that because the goroutine is only created when the deferred function needs to be executed, this method does not waste resources.

### Cache index 

A global cache index is also used, which is a map associating cache keys with their respective deletion times. After a cache entry was created and written to, the key and deletion time pair gets added to the index, and the key gets removed before the entry is deleted. Without such an index, the only way to determine whether a client request was already cached is to make a system call to determine if the associated file is existent. This is a waste of resources which can easily be avoided. 

### Persistence

This HTTP Proxy is able to persist its cache even if it goes down for some time. The persistence is accomplished through [Docker volumes](https://docs.docker.com/storage/volumes/).

The coexistence of persistence and a [cache index](#cache-index) has to be dealt with. That is, the cache index, which is in-memory, also has to be correctly persisted.

When the application receives a shutdown signal, it will encode the cache index to file before it terminates.

Here is what happens when the application is relaunched:

- The index cache is decoded from the saved file into a temporary map
- For each `key:deletion time` pair in the temporary map
  - if the deletion time is in the past, delete the associated cache file and do not add the pair to the index
  - otherwise, add the pair to the index, and reschedule the deletion of the cache file (each cache file is scheduled for deletion; see [How is the cache actually implemented?](#how-is-the-cache-actually-implemented) above).

### Prioritize serving over caching

Even though this proxy's purpose is to cache HTTP responses, it should still serve requests as fast as possible and not let caching delay response time. This is why it was decided, when an upstream response is cacheable, to serve the response and cache it concurrently, but respond to the client as quickly as possible.

That being said, the upstream response body is a stream, meaning that once it is read from, if the bytes read were not stored, they are gone and cannot be read a second time. There are at least two ways to deal with this while observing the aforementioned constraint.

The first way is to use Go's [io.TeeReader](https://pkg.go.dev/io#TeeReader) in combination with [io.Pipe](https://pkg.go.dev/io#Pipe). The goroutine responsible for responding to the client will access the body from the tee reader, which means that as the upstream response body is read, its contents will also be written to a writer – this is what io.TeeReader does. If that writer is the write half of an io.Pipe, we can pass the read half of the pipe to the goroutine responsible for caching the upstream response, and that's how it will also have access to the response body. However, an important thing to note here is that the pipe is synchronous, meaning that data has to be read from the read half as it is written to the write half, because no internal buffering takes place.

The second way to deal with the streaming nature of the response body involves once again [io.TeeReader](https://pkg.go.dev/io#TeeReader), but this time used with a simple byte buffer. Again, the goroutine responsible for responding to the client will access the body from the tee reader. But this time around, as the response body is read, its contents are written to the byte buffer. Because that buffer can be read from in addition to written to, it can directly be passed to the caching goroutine (without the need for a pipe). 

The first solution is arguably more elegant and does not use more memory. However, because the write and read operations of the pipe are synchronous, this means that the response cannot be served to the client until the cache file is fully written to. This seems like an unnecessary delay.

The second solution does consume memory, in order to store the response body in a buffer. But, with this solution, the response goroutine has no temporal dependency on the caching goroutine (the response is never going to wait for caching to proceed), which seems preferable.

We elected to go with the second solution (see function serveFromUpstream in internal/proxy.go), pending a better, more refined approach.

### How are headers from the upstream treated?

Only the following headers from upstream are kept:
- Content-Type
- Cache-Control
- Date
- Expired
- Set-Cookie

A custom server header is also added to all responses.

What's more, a response sent from the cache will have a `X-Cache: HIT` header and an `Age` header specifying the number of seconds elapsed since the request was committed to cache.

A response sent directly from the upstream server will have a `X-Cache: MISS` header.
