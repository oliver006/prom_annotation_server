# Promdash Annotation Server

An Annotation Server for [PromDash](https://github.com/prometheus/promdash) (the [Prometheus](https://github.com/prometheus/prometheus) Dashboard Builder)


## Building and running

    go build
    ./prom_annotation_server <flags>


### Flags

Name               | Description
-------------------|------------
storage            | Storage config, format is `type:options`. *local* is currently the only supported type with options being the location of the DB file. Example: *local:/tmp/annotations.db* 
listen-addr        | Address to listen on, defaults to `:9119`
endpoint           | Path under which to expose the annotation server, defaults to `/annotations`
version            | Show version information and exit


### How to add annotations and configure PromDash?

Once the annotation server is up and running you can add annotations by making HTTP PUT requests to the configured endpoint (default: `:9119/annotations`):
```
$ curl -XPUT -d '{"message":"build: web server", "tags": ["build"] }'  "localhost:9119/annotations"
{"result":"ok"}
$
```

This will add a tag with the current timestamp to all tags listed in *tags*.<br>
You can also provide the timestamp:
```
curl -XPUT -d '{"created_at": 1430797123000, "message":"build: web server", "tags": ["build"] }'  "localhost:9119/annotations"
```
and add an annotation for multiple tags:
```
curl -XPUT -d '{"created_at": 1430797123000, "message":"build: web server", "tags": ["build-prod", "build-dev"] }'  "localhost:9119/annotations"
```

To make PromDash pick up annotations, you need to set the `ANNOTATIONS_URL` to e.g. `http://localhost:9119/annotations` before starting promdash. See the [official docs here](http://prometheus.io/docs/visualization/promdash/#annotations) for more detailed information.


You can also query the annotation server yourself:
```
$ curl 'localhost:9119/annotations?tags\[\]=build'
{"posts":[{"created_at":1430797123000,"message":"build: web server"},{"created_at":1430797150000,"message":"build: web server"}]}
```

By default, the annotation server will show tags for the last 3600 seconds from now on but you can also override the filters by providing the `until` (absolute timestamp) and `r` (for range) parameters, both in seconds.

### Hmmmkay, but where do you store my data?

Right now, the annotation server supports local storage on disk (using [BoltDB](https://github.com/boltdb/bolt) for the storage engine) and (RethinkDB)[http://rethinkdb.com/] (with more to come!)<br>

By default, the annotation server will use the *local* option with `/tmp/annotations.db` the default location to store annotations but by using the `--storage` parameter you can provide different options.<br>
 
Local example:
`./prom_annotation_server --storage=local:/data/prometheus/annotations.db`

RethinkDB example:
`./prom_annotation_server --storage=rethinkdb:localhost:28015/annotations`
where `localhost:28015` is the host name to connect to and `annotations` after the slash is the name of ht eB to use. If the DB doesn't exist it's created along with the table `annotations` that holds the annotations.

Adding a new storage provider is easy. I you're interested in adding a new storage engine then have a look at [storage_boltdb.go](blob/master/storage_boltdb.go) or [storage_rethinkdb.go](blob/master/storage_rethinkdb.go) to see what's needed, it's very straight forward.


### Cool, what's next?

- some sort of tag management endpoint would be nice to support deleting of tags
- more storage providers
- make the annotation server export its own set of metrics
- ...



